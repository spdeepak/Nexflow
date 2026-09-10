// --- Chat ---

const CHAT_EVENT = 'chat:event';
const CHAT_INTERRUPT_EVENT = 'chat:interrupt';

let chatSessionsData = [];
let chatAgentList = [];
let chatTreeAgents = [];   // [{ id, name, kind: 'parent'|'sub', position }]
let chatSessionId = '';
let chatRunning = false;
let chatAllEvents = [];        // all persisted events for the open session (cached)
let chatHistoryReplay = null;  // { events:[...], order:[agentName,...] } for the active replay
let chatReplayTimer = null;

async function renderChat(container) {
    chatSessionsData = await window.go.application.App.ListSessions();
    chatAgentList = await window.go.application.App.GetAgents();

    container.innerHTML = `
        <div class="page-header">
            <h1>Chat</h1>
        </div>
        <div class="chat-shell">
            <div class="chat-sidebar" id="chat-sidebar">
                <button class="btn btn-primary btn-small" onclick="newChat()">New Chat</button>
                <div class="chat-session-list" id="chat-session-list"></div>
            </div>
            <div class="chat-view" id="chat-view"></div>
        </div>`;

    document.getElementById('chat-sidebar').addEventListener('click', (e) => {
        if (e.target.closest('.chat-session-item') || e.target.closest('button')) return;
        if (e.target.id === 'chat-sidebar' || e.target.id === 'chat-session-list') {
            deselectChat();
        }
    });

    renderChatSessionList();

    if (chatSessionId && chatSessionsData.some(s => s.id === chatSessionId)) {
        openChat(chatSessionId);
    } else {
        showChatPlaceholder();
    }
}

function showChatPlaceholder() {
    const view = document.getElementById('chat-view');
    if (view) view.innerHTML = '<div class="chat-placeholder">Select a conversation</div>';
}

function deselectChat() {
    clearEventHistory();
    unbindChatEvents();
    chatSessionId = '';
    showChatPlaceholder();
    renderChatSessionList();
}

function renderChatSessionList() {
    const list = document.getElementById('chat-session-list');
    if (!list) return;
    list.innerHTML = '';

    if (chatSessionsData.length === 0) {
        list.innerHTML = '<div class="empty-state">No conversations yet. Start a new chat.</div>';
        return;
    }

    for (const s of chatSessionsData) {
        const rootName = chatAgentList.find(a => a.id === s.rootAgentId)?.name || s.rootAgentId;
        const item = document.createElement('div');
        item.className = 'chat-session-item' + (s.id === chatSessionId ? ' active' : '');
        item.onclick = () => openChat(s.id);
        const updated = s.lastUpdate ? new Date(s.lastUpdate).toLocaleString() : '';
        item.innerHTML = `
            <div class="chat-session-info">
                <div class="chat-session-title">${escapeHtml(rootName)}</div>
                <div class="chat-session-meta">${escapeHtml(updated)}</div>
            </div>
            <button class="chat-session-delete" onclick="event.stopPropagation(); deleteChat('${s.id}')" title="Delete">&times;</button>`;
        list.appendChild(item);
    }
}

function newChat() {
    const active = chatAgentList.filter(a => a.isActive);
    if (active.length === 0) {
        showToast('Create an agent first');
        return;
    }

    let options = '';
    for (const a of active) {
        options += `<option value="${a.id}">${escapeHtml(a.name)}</option>`;
    }

    showModal(
        'New Chat',
        `<div class="form-group">
            <label>Agent</label>
            <select id="new-chat-agent">${options}</select>
        </div>`,
        async () => {
            const rootAgentId = document.getElementById('new-chat-agent').value;
            try {
                const session = await window.go.application.App.CreateSession(rootAgentId);
                chatSessionsData.unshift(session);
                hideModal();
                openChat(session.id, rootAgentId);
            } catch (e) {
                showToast('Failed to create chat: ' + e);
            }
        },'Start Chat'
    );
}

function deleteChat(sessionId) {
    window.go.application.App.DeleteSession(sessionId).then(() => {
        chatSessionsData = chatSessionsData.filter(s => s.id !== sessionId);
        if (chatSessionId === sessionId) {
            chatSessionId = '';
            clearEventHistory();
            unbindChatEvents();
            showChatPlaceholder();
        }
        renderChatSessionList();
    }).catch(e => showToast('Failed to delete: ' + e));
}

async function openChat(sessionId, rootAgentId) {
    chatSessionId = sessionId;
    const view = document.getElementById('chat-view');
    if (!view) return;

    let detail;
    const agentId = rootAgentId || chatSessionsData.find(s => s.id === sessionId)?.rootAgentId || sessionId;
    try {
        detail = await window.go.application.App.GetAgent(agentId);
    } catch (e) {
        detail = { name: 'Unnamed' };
    }

    let subAgents = [];
    if (detail.id) {
        try {
            subAgents = await window.go.application.App.GetSubAgents(detail.id);
            subAgents = subAgents.filter(s => s.isActive);
        } catch (e) {
            subAgents = [];
        }
    }

    chatTreeAgents = [
        { id: detail.id || agentId, name: detail.name || 'Parent', kind: 'parent', position: -1 },
        ...subAgents.map((s, i) => ({ id: s.id, name: s.name || s.id, kind: 'sub', position: i }))
    ];

    view.innerHTML = `
        <div class="chat-header">
            <h2>${escapeHtml(detail.name || 'Chat')}</h2>
        </div>
        <div class="chat-layout">
            <div class="chat-tree" id="chat-tree"></div>
            <div class="chat-main">
                <div class="chat-history" id="chat-history"></div>
                <div class="chat-input">
                    <textarea id="chat-msg" rows="2" placeholder="Ask your agents..."></textarea>
                    <button class="btn btn-primary" id="chat-send" onclick="sendChatMessage()">Send</button>
                </div>
            </div>
        </div>`;

    renderChatTree();
    document.getElementById('chat-msg').addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendChatMessage();
        }
    });

    unbindChatEvents();
    window.runtime.EventsOn(CHAT_EVENT, onChatEvent);
    window.runtime.EventsOn(CHAT_INTERRUPT_EVENT, onChatInterrupt);
    await loadChatHistory();
    renderChatSessionList();
}

function renderChatTree() {
    const tree = document.getElementById('chat-tree');
    let html = '<div class="tree-title">Agents</div>';

    for (let i = 0; i < chatTreeAgents.length; i++) {
        const node = chatTreeAgents[i];
        const kindClass = node.kind === 'parent' ? 'node-parent' : 'node-sub';
        const status = node.responseText ? '' : 'state-idle';
        if (i > 0) html += '<div class="tree-connector"></div>';
        html += `
            <div class="tree-node ${kindClass} ${status}" id="tree-node-${i}" data-index="${i}">
                <div class="tree-node-body" onclick="selectTreeNode(${i})">
                    <div class="tree-node-name">${escapeHtml(node.name)}</div>
                    <div class="tree-node-status" id="tree-status-${i}">${kindClass === 'node-parent' ? 'Parent' : 'Sub-agent ' + (node.position + 1)}</div>
                </div>
            </div>`;
    }

    tree.innerHTML = html;
}

function selectTreeNode(index) {
    const node = chatTreeAgents[index];
    const panel = new EventDetail(node);
    if (chatHistoryReplay) {
        panel.setHistoryText(chatHistoryReplay.byAgent[node.name] || chatHistoryReplay.byAgent[node.id] || '');
    }
    panel.show();
}

// --- Event history replay ---

function showEventHistory(invocationId) {
    clearEventHistory();

    const runEvents = chatAllEvents.filter(ev => ev.invocationId === invocationId);
    if (runEvents.length === 0) return;

    const participants = [];
    const byAgent = {};
    for (const ev of runEvents) {
        if (ev.role !== 'model' && ev.role !== 'assistant' && ev.role !== 'function') continue;
        if (participants.indexOf(ev.author) === -1) participants.push(ev.author);
        const text = extractText(ev);
        if (text) byAgent[ev.author] = (byAgent[ev.author] || '') + text + '\n\n';
    }

    chatHistoryReplay = { events: runEvents, order: participants, byAgent };

    const tree = document.getElementById('chat-tree');
    tree.scrollTop = 0;
    let i = 0;
    const step = () => {
        if (i >= participants.length) {
            const lastAgent = participants[participants.length - 1];
            const lastIdx = chatTreeAgents.findIndex(n => n.name === lastAgent || n.id === lastAgent);
            if (lastIdx >= 0) {
                setNodeState(document.getElementById(`tree-node-${lastIdx}`), 'state-done');
            }
            chatReplayTimer = null;
            return;
        }
        const name = participants[i];
        const idx = chatTreeAgents.findIndex(n => n.name === name || n.id === name);
        for (let k = 0; k < i; k++) {
            const prevIdx = chatTreeAgents.findIndex(n => n.name === participants[k] || n.id === participants[k]);
            if (prevIdx >= 0) {
                setNodeState(document.getElementById(`tree-node-${prevIdx}`), 'state-done');
            }
        }
        if (idx >= 0) {
            const el = document.getElementById(`tree-node-${idx}`);
            if (el) {
                setNodeState(el, 'state-running');
                el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
            }
        }
        i++;
        chatReplayTimer = setTimeout(step, 500);
    };
    step();
}

function resetTreeGlow() {
    for (let i = 0; i < chatTreeAgents.length; i++) {
        const el = document.getElementById(`tree-node-${i}`);
        if (el) el.classList.remove('state-running', 'state-done');
    }
}

function clearEventHistory() {
    if (chatReplayTimer) { clearTimeout(chatReplayTimer); chatReplayTimer = null; }
    chatHistoryReplay = null;
    resetTreeGlow();
}

async function loadChatHistory() {
    const history = document.getElementById('chat-history');
    history.innerHTML = '<div class="loading">Loading...</div>';
    let events;
    try {
        events = await window.go.application.App.ListEvents(chatSessionId);
    } catch (e) {
        history.innerHTML = '<div class="empty-state">Failed to load history</div>';
        return;
    }

    const sess = chatSessionsData.find(s => s.id === chatSessionId);
    const rootAgent = chatAgentList.find(a => a.id === sess?.rootAgentId);
    const rootName = rootAgent?.name;
    const rootId = rootAgent?.id;

    chatAllEvents = events;
    history.innerHTML = '';
    for (const ev of events) {
        if (ev.role === 'user') {
            appendBubble(history, 'user', extractText(ev), ev.createdAt);
        } else if (ev.role === 'model' || ev.role === 'assistant') {
            if ((rootName || rootId) && ev.author && ev.author !== rootName && ev.author !== rootId) continue;
            const text = extractText(ev);
            if (text) {
                appendBubble(history, 'agent', text, ev.createdAt, ev.invocationId);
            }
        }
    }
}

function appendBubble(container, role, text, dateStr, invocationId) {
    const wrap = document.createElement('div');
    wrap.className = `chat-message chat-message-${role}`;

    const el = document.createElement('div');
    el.className = `chat-bubble bubble-${role} md-body`;
    el.innerHTML = markdownToHtml(text || '');
    wrap.appendChild(el);

    const meta = document.createElement('div');
    meta.className = 'bubble-time';
    const timeSpan = document.createElement('span');
    timeSpan.textContent = formatBubbleTime(dateStr);
    meta.appendChild(timeSpan);

    if (role === 'agent' && invocationId) {
        const btn = document.createElement('button');
        btn.className = 'bubble-event-btn';
        btn.textContent = 'Event history';
        btn.title = 'Replay agent event flow for this response';
        btn.onclick = () => showEventHistory(invocationId);
        meta.appendChild(btn);
    }
    wrap.appendChild(meta);

    container.appendChild(wrap);
    container.scrollTop = container.scrollHeight;
}

function formatBubbleTime(dateStr) {
    const d = dateStr ? new Date(dateStr) : new Date();
    if (isNaN(d.getTime())) return '';
    const now = new Date();
    const sameDay = d.getFullYear() === now.getFullYear() &&
        d.getMonth() === now.getMonth() &&
        d.getDate() === now.getDate();
    const time = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    return sameDay ? time : `${d.toLocaleDateString()} ${time}`;
}

function extractText(event) {
    if (!event) return '';
    try {
        let parts;
        if (event.contentJson) {
            const content = typeof event.contentJson === 'string' ? JSON.parse(event.contentJson) : event.contentJson;
            parts = content.parts || content;
        } else if (event.content && event.content.parts) {
            parts = event.content.parts;
        }
        if (!parts || !Array.isArray(parts)) return '';
        return parts.filter(p => p && p.text).map(p => p.text).join('\n');
    } catch (e) {
        return '';
    }
}

async function sendChatMessage() {
    const input = document.getElementById('chat-msg');
    const msg = input.value.trim();
    if (!msg || chatRunning) return;

    input.value = '';
    clearEventHistory();
    const history = document.getElementById('chat-history');
    appendBubble(history, 'user', msg, new Date().toISOString());
    setChatRunning(true);
    resetTreeStates();

    try {
        await window.go.application.App.SendMessageSession(chatSessionId, msg);
        await loadChatHistory();
    } catch (e) {
        showToast('Failed: ' + e);
    } finally {
        setChatRunning(false);
        clearAllTreeStates();
    }
}

function setChatRunning(running) {
    chatRunning = running;
    const sendBtn = document.getElementById('chat-send');
    if (sendBtn) sendBtn.disabled = running;
}

function setNodeState(el, state) {
    if (!el) return;
    el.classList.remove('state-running', 'state-done', 'state-idle');
    if (state) el.classList.add(state);
}

function resetTreeStates() {
    for (let i = 0; i < chatTreeAgents.length; i++) {
        chatTreeAgents[i].responseText = '';
        const el = document.getElementById(`tree-node-${i}`);
        if (el) {
            setNodeState(el, 'state-idle');
        }
    }
}

function clearAllTreeStates() {
    for (let i = 0; i < chatTreeAgents.length; i++) {
        const el = document.getElementById(`tree-node-${i}`);
        if (el) {
            if (el.classList.contains('state-running')) {
                setNodeState(el, 'state-done');
                const st = document.getElementById(`tree-status-${i}`);
                if (st) st.textContent = 'Done';
            } else {
                el.classList.remove('state-running');
            }
        }
    }
}

function onChatEvent(payload) {
    if (!payload) return;

    const ev = payload.event;
    if (ev) {
        window.__chatTrace = window.__chatTrace || [];
        window.__chatTrace.push({
            agent: payload.agent, author: ev.author, final: payload.final,
            partial: ev.partial, role: ev.content && ev.content.role
        });
        console.debug('[chat:event]', window.__chatTrace[window.__chatTrace.length - 1]);
    }

    if (!ev) {
        applyFinalState(payload);
        return;
    }

    if (ev.author === 'user') {
        if (payload.final) applyFinalState(payload);
        return;
    }

    // Match either by agent ID or agent Name
    const idx = chatTreeAgents.findIndex(node => node.name === payload.agent || node.id === payload.agent);
    if (idx >= 0) {
        for (let i = 0; i < chatTreeAgents.length; i++) {
            const el = document.getElementById(`tree-node-${i}`);
            if (!el) continue;
            if (i === idx) {
                // Active agent pulses
                setNodeState(el, 'state-running');
            } else if (el.classList.contains('state-done') || i < idx) {
                // Previously active agents stay or become green (done)
                setNodeState(el, 'state-done');
            } else {
                setNodeState(el, 'state-idle');
            }
        }

        const text = extractText(ev);
        if (text) chatTreeAgents[idx].responseText += text;
    }

    if (payload.final) {
        applyFinalState(payload);
    }
}

function applyFinalState(payload) {
    const idx = chatTreeAgents.findIndex(node => node.name === payload.agent || node.id === payload.agent);
    for (let i = 0; i < chatTreeAgents.length; i++) {
        const el = document.getElementById(`tree-node-${i}`);
        if (!el) continue;
        el.classList.remove('state-running');
    }
    if (idx >= 0) {
        const el = document.getElementById(`tree-node-${idx}`);
        if (el) {
            setNodeState(el, 'state-done');
            const st = document.getElementById(`tree-status-${idx}`);
            if (st) st.textContent = 'Done';
        }
    }
}

// --- Human-in-the-Loop ---

function onChatInterrupt(interrupt) {
    if (!interrupt || !interrupt.callId) return;
    setPendingConfirmation(interrupt);
    if (!chatRunning) chatRunning = true;

    const history = document.getElementById('chat-history');
    if (!history) return;

    const toolName = interrupt.toolName || 'the tool';
    const hint = interrupt.hint || `Approve ${toolName}?`;

    const wrap = document.createElement('div');
    wrap.className = 'chat-message chat-message-agent';
    wrap.id = 'chat-interrupt-card';
    wrap.innerHTML = `
        <div class="chat-bubble bubble-agent chat-confirm">
            <div class="chat-confirm-title">Approval Required</div>
            <div class="chat-confirm-desc">${escapeHtml(hint)}</div>
            <div class="chat-confirm-tool">Tool: <code>${escapeHtml(toolName)}</code></div>
            <textarea id="chat-confirm-note" rows="2" placeholder="Add extra info (optional)"></textarea>
            <div class="chat-confirm-actions">
                <button class="btn btn-primary btn-small" onclick="resumeChat(true, this)">Approve</button>
                <button class="btn btn-small" onclick="resumeChat(false, this)">Reject</button>
            </div>
        </div>
        <div class="bubble-time"><span>Awaiting your decision…</span></div>`;
    history.appendChild(wrap);
    history.scrollTop = history.scrollHeight;
}

async function resumeChat(confirmed, btn) {
    if (!btn) return;
    const callID = window.__pendingConfirmationCallId;
    const runID = window.__pendingConfirmationRunId;
    if (!callID) return;

    btn.disabled = true;
    const actions = btn.closest('.chat-confirm-actions');
    if (actions) {
        for (const b of actions.querySelectorAll('button')) b.disabled = true;
    }

    const noteEl = document.getElementById('chat-confirm-note');
    const note = noteEl ? noteEl.value.trim() : '';

    try {
        await window.go.application.App.ConfirmSession(chatSessionId, callID, confirmed, note);
    } catch (e) {
        showToast('Failed to resume: ' + e);
    }

    window.__pendingConfirmationCallId = '';
    window.__pendingConfirmationRunId = '';
    removePendingConfirmationCard();

    if (chatRunning) setChatRunning(false);
    await loadChatHistory();
    clearAllTreeStates();
}

function removePendingConfirmationCard() {
    const card = document.getElementById('chat-interrupt-card');
    if (card) card.remove();
}

function setPendingConfirmation(interrupt) {
    window.__pendingConfirmationCallId = interrupt.callId;
    window.__pendingConfirmationRunId = interrupt.runId || '';
}

// --- Node detail panel ---

class EventDetail {
    constructor(node) {
        this.node = node;
        this.historyText = '';
    }
    setHistoryText(text) {
        this.historyText = text;
    }
    show() {
        const kindLabel = this.node.kind === 'parent' ? 'Parent Agent' : 'Sub-agent';
        const kindClass = this.node.kind === 'parent' ? 'detail-parent' : 'detail-sub';
        const overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        const bodyText = this.historyText || this.node.responseText || 'No response yet';
        const badge = this.historyText ? `${kindLabel} &middot; Event history` : kindLabel;
        overlay.innerHTML = `
            <div class="detail-panel ${kindClass}">
                <div class="detail-header">
                    <div class="detail-header-left">
                        <span class="detail-agent-icon">${this.node.kind === 'parent' ? '&#9670;' : '&#9671;'}</span>
                        <div>
                            <div class="detail-agent-name">${escapeHtml(this.node.name)}</div>
                            <span class="detail-agent-badge">${badge}</span>
                        </div>
                    </div>
                    <span class="modal-close" onclick="this.closest('.modal-overlay').remove()">&#10005;</span>
                </div>
                <div class="detail-body">
                    <div class="detail-response md-body">${markdownToHtml(bodyText)}</div>
                </div>
            </div>`;
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) overlay.remove();
        });
        document.body.appendChild(overlay);
    }
}

function unbindChatEvents() {
    try { window.runtime.EventsOff(CHAT_EVENT); } catch (e) {}
    try { window.runtime.EventsOff(CHAT_INTERRUPT_EVENT); } catch (e) {}
}

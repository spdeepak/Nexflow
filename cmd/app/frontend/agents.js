// --- Agents ---

let agentListData = [];
let modelOptions = [];
let currentSubAgents = [];
let currentAgentId = '';
let agentNavStack = [];

async function renderAgents(container) {
    agentListData = await window.go.application.App.GetAgents();
    agentNavStack = [];

    renderAgentList(container);
}

function renderAgentList(container) {
    let html = `
        <div class="page-header">
            <h1>Agents</h1>
            <button class="btn btn-primary" onclick="showCreateAgentModal()">Create Agent</button>
        </div>`;

    if (agentListData.length === 0) {
        html += '<div class="empty-state">No agents found</div>';
    } else {
        html += '<div class="cards-grid">';
        for (const a of agentListData) {
            const statusClass = a.isActive ? 'status-active' : 'status-inactive';
            const statusText = a.isActive ? 'Active' : 'Inactive';

            html += `
                <div class="card" onclick="openRootAgentDetail('${a.id}')" style="cursor:pointer">
                    <div class="card-title">${escapeHtml(a.name || a.id)}</div>
                    <div class="card-body">
                        ${a.description ? `<div class="card-label">${escapeHtml(a.description)}</div>` : ''}
                        <div class="card-subtitle">Mode: ${escapeHtml(a.mode)}</div>
                        ${a.modelName ? `<div class="card-label">Model: <strong>${escapeHtml(a.modelName)}</strong></div>` : ''}
                        <div class="card-label">Credential: ${escapeHtml(a.credentialSource)}</div>
                        <div><span class="status ${statusClass}">${statusText}</span></div>
                    </div>
                </div>`;
        }
        html += '</div>';
    }

    container.innerHTML = html;
}

async function showAgentDetail(agentId) {
    const container = document.getElementById('view-container');
    currentAgentId = agentId;

    let detail;
    try {
        detail = await window.go.application.App.GetAgent(agentId);
    } catch (e) {
        console.error('Failed to load agent:', e);
        showToast('Failed to load agent details');
        return;
    }

    const statusClass = detail.isActive ? 'status-active' : 'status-inactive';
    const statusText = detail.isActive ? 'Active' : 'Inactive';

    let subAgentsHtml = '';
    try {
        currentSubAgents = await window.go.application.App.GetSubAgents(agentId);
        if (currentSubAgents.length > 0) {
            subAgentsHtml = '<div class="kb-list" style="margin-top:16px">';
            for (let i = 0; i < currentSubAgents.length; i++) {
                subAgentsHtml += renderSubAgentRow(currentSubAgents[i], i, currentSubAgents.length);
            }
            subAgentsHtml += '</div>';
        } else {
            subAgentsHtml = '<div class="empty-state" style="padding:20px">No sub-agents</div>';
        }
    } catch (e) {
        console.error('Failed to load sub-agents:', e);
    }

    let html = `
        <div class="page-header">
            <div style="display:flex;align-items:center;gap:8px">
                <button class="skill-back-btn" onclick="goBackInAgentStack()" title="Back">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(detail.name)}</h1>
            </div>
            <div style="display:flex;gap:8px">
                <button class="btn btn-primary btn-small" onclick="showCreateSubAgentModal('${detail.id}', '${escapeHtml(detail.name)}')">Create Sub Agent</button>
                <button class="btn btn-small" onclick="showEditAgentModal('${detail.id}')">Edit</button>
                <button class="btn btn-small" style="background:#e74c3c;color:#fff" onclick="confirmDeleteAgent('${detail.id}', '${escapeHtml(detail.name)}')">Delete</button>
            </div>
        </div>
        <div class="skill-detail-card">
            <div class="skill-detail-meta">
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Mode</span>
                    <span class="skill-detail-meta-value">${escapeHtml(detail.mode)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Model</span>
                    <span class="skill-detail-meta-value">${escapeHtml(detail.modelName)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Status</span>
                    <span class="skill-detail-meta-value">
                        <span class="status ${statusClass}">${statusText}</span>
                    </span>
                </div>
            </div>
            ${detail.description ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Description</div>
                <pre class="skill-detail-content">${escapeHtml(detail.description)}</pre>
            </div>` : ''}
            ${detail.instruction ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Instruction</div>
                <pre class="skill-detail-content">${escapeHtml(detail.instruction)}</pre>
            </div>` : ''}
            ${detail.globalInstruction ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Global Instruction</div>
                <pre class="skill-detail-content">${escapeHtml(detail.globalInstruction)}</pre>
            </div>` : ''}
        </div>
        <div style="margin-top:24px">
            <h2 style="font-size:18px;font-weight:700;margin-bottom:12px">Sub-Agents</h2>
            <div id="sub-agents-container">${subAgentsHtml}</div>
        </div>
        <div style="margin-top:24px">
            <h2 style="font-size:18px;font-weight:700;margin-bottom:12px">Skills</h2>
            ${renderSkillsSection(detail.skills || [])}
        </div>
        <div style="margin-top:24px">
            <h2 style="font-size:18px;font-weight:700;margin-bottom:12px">MCP Servers</h2>
            ${renderMcpsSection(detail.mcps || [])}
        </div>`;

    container.innerHTML = html;
    attachSubAgentDragHandlers(container);
}

function confirmDeleteAgent(agentId, agentName) {
    const body = `<p>Are you sure you want to delete the agent "${escapeHtml(agentName)}"?</p>`;
    showModal('Delete Agent', body, () => deleteAgent(agentId), 'Delete');
}

async function deleteAgent(agentId) {
    try {
        await window.go.application.App.DeleteAgent(agentId);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to delete agent: ' + err);
    }
}

let editModelOptions = [];

async function showEditAgentModal(agentId) {
    let detail;
    try {
        detail = await window.go.application.App.GetAgent(agentId);
    } catch (e) {
        showToast('Failed to load agent details');
        return;
    }

    editModelOptions = await window.go.application.App.GetModelOptions();
    const modelCredentialId = detail.modelCredentialId || '';

    const body = `
        <div class="form-group">
            <label>Name</label>
            <input type="text" value="${escapeHtml(detail.name)}" disabled>
        </div>
        <div class="form-group">
            <label for="edit-agent-desc">Description</label>
            <textarea id="edit-agent-desc">${escapeHtml(detail.description || '')}</textarea>
        </div>
        <div class="form-group">
            <label for="edit-agent-instr">Instructions</label>
            <textarea id="edit-agent-instr">${escapeHtml(detail.instruction || '')}</textarea>
        </div>
        <div class="form-group">
            <label for="edit-agent-ginstr">Global Instructions</label>
            <textarea id="edit-agent-ginstr">${escapeHtml(detail.globalInstruction || '')}</textarea>
        </div>
        <div class="form-group">
            <label for="edit-agent-mode">Mode</label>
            <select id="edit-agent-mode">
                <option value="chat" ${detail.mode === 'chat' ? 'selected' : ''}>chat</option>
                <option value="task" ${detail.mode === 'task' ? 'selected' : ''}>task</option>
                <option value="single_turn" ${detail.mode === 'single_turn' ? 'selected' : ''}>single_turn</option>
            </select>
        </div>
        <div class="form-group">
            <label for="edit-agent-model">Model</label>
            <select id="edit-agent-model">
                <option value="">-- Select Model --</option>
                ${editModelOptions.map(m => `<option value="${m.value}" ${m.value === modelCredentialId ? 'selected' : ''}>${escapeHtml(m.label)}</option>`).join('')}
            </select>
        </div>
        <div class="form-group">
            <label>Credential Source</label>
            <input type="text" value="${escapeHtml(detail.credentialSource)}" disabled>
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="edit-agent-active" ${detail.isActive ? 'checked' : ''}> Active</label>
        </div>
        <div class="accordion">
            <button type="button" class="accordion-toggle" onclick="toggleAccordion(this)">
                Model Configuration (Advanced) <span class="chevron">&#9654;</span>
            </button>
            <div class="accordion-content">
                <div class="form-group">
                    <label for="mc-temperature">Temperature (0.0 - 2.0)</label>
                    <input type="number" id="mc-temperature" step="0.1" min="0" max="2" placeholder="e.g. 0.7" value="${detail.modelConfig?.temperature ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-topP">Top P</label>
                    <input type="number" id="mc-topP" step="0.05" min="0" max="1" placeholder="e.g. 0.95" value="${detail.modelConfig?.top_p ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-topK">Top K</label>
                    <input type="number" id="mc-topK" min="1" placeholder="e.g. 40" value="${detail.modelConfig?.top_k ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-maxTokens">Max Output Tokens</label>
                    <input type="number" id="mc-maxTokens" min="1" placeholder="e.g. 1024" value="${detail.modelConfig?.max_output_tokens ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-frequencyPenalty">Frequency Penalty</label>
                    <input type="number" id="mc-frequencyPenalty" step="0.1" placeholder="e.g. 0.0" value="${detail.modelConfig?.frequency_penalty ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-presencePenalty">Presence Penalty</label>
                    <input type="number" id="mc-presencePenalty" step="0.1" placeholder="e.g. 0.0" value="${detail.modelConfig?.presence_penalty ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-seed">Seed</label>
                    <input type="number" id="mc-seed" min="0" placeholder="e.g. 42" value="${detail.modelConfig?.seed ?? ''}">
                </div>
                <div class="form-group">
                    <label for="mc-stopSequences">Stop Sequences (comma separated)</label>
                    <input type="text" id="mc-stopSequences" placeholder="e.g. END, STOP" value="${(detail.modelConfig?.stop_sequences || []).join(', ')}">
                </div>
                <div class="form-group">
                    <label for="mc-responseMimeType">Response MIME Type</label>
                    <select id="mc-responseMimeType">
                        <option value="">-- Default --</option>
                        <option value="application/json" ${detail.modelConfig?.response_mime_type === 'application/json' ? 'selected' : ''}>application/json</option>
                        <option value="text/plain" ${detail.modelConfig?.response_mime_type === 'text/plain' ? 'selected' : ''}>text/plain</option>
                    </select>
                </div>
                <div class="form-group">
                    <label for="mc-responseLogprobs">
                        <input type="checkbox" id="mc-responseLogprobs" style="width:auto;margin-right:6px;" ${detail.modelConfig?.response_logprobs ? 'checked' : ''}> Return Log Probs
                    </label>
                </div>
                <div class="form-group">
                    <label for="mc-audioTimestamp">
                        <input type="checkbox" id="mc-audioTimestamp" style="width:auto;margin-right:6px;" ${detail.modelConfig?.audio_timestamp ? 'checked' : ''}> Audio Timestamp
                    </label>
                </div>
            </div>
        </div>`;

    showModal('Edit Agent', body, () => submitEditAgent(agentId),'Update');
}

async function submitEditAgent(agentId) {
    const modelSelect = document.getElementById('edit-agent-model');
    const modelCredentialId = modelSelect.value;
    const selectedModel = editModelOptions.find(m => m.value === modelCredentialId);
    const modelMatch = selectedModel?.label.match(/\(([^/]+)\/([^)]+)\)/);

    const params = {
        description: document.getElementById('edit-agent-desc').value,
        instruction: document.getElementById('edit-agent-instr').value,
        globalInstruction: document.getElementById('edit-agent-ginstr').value,
        mode: document.getElementById('edit-agent-mode').value,
        modelName: modelMatch ? modelMatch[2] : '',
        modelCredentialId: modelCredentialId,
        credentialSource: 'app',
        isActive: document.getElementById('edit-agent-active').checked,
        modelConfig: buildModelConfig(),
    };

    try {
        await window.go.application.App.UpdateAgent(agentId, params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to update agent: ' + err);
    }
}

let subAgentModelOptions = [];

async function showCreateSubAgentModal(parentId, parentName) {
    subAgentModelOptions = await window.go.application.App.GetModelOptions();

    const body = `
        <div class="form-group">
            <label>Parent Agent</label>
            <input type="text" value="${escapeHtml(parentName)}" disabled>
        </div>
        <div class="form-group">
            <label for="sub-agent-name">Name *</label>
            <input type="text" id="sub-agent-name" placeholder="Sub-agent name" required>
        </div>
        <div class="form-group">
            <label for="sub-agent-desc">Description</label>
            <textarea id="sub-agent-desc" placeholder="Description"></textarea>
        </div>
        <div class="form-group">
            <label for="sub-agent-instr">Instructions *</label>
            <textarea id="sub-agent-instr" placeholder="Instructions for the sub-agent" required></textarea>
        </div>
        <div class="form-group">
            <label for="sub-agent-ginstr">Global Instructions</label>
            <textarea id="sub-agent-ginstr" placeholder="Global instructions"></textarea>
        </div>
        <div class="form-group">
            <label for="sub-agent-mode">Mode</label>
            <select id="sub-agent-mode" required>
                <option value="chat">chat</option>
                <option value="task">task</option>
                <option value="single_turn">single_turn</option>
            </select>
        </div>
        <div class="form-group">
            <label for="sub-agent-model">Model *</label>
            <select id="sub-agent-model" required>
                <option value="">-- Select Model --</option>
                ${subAgentModelOptions.map(m => `<option value="${m.value}">${escapeHtml(m.label)}</option>`).join('')}
            </select>
        </div>`;

    showModal('Create Sub Agent', body, () => submitCreateSubAgent(parentId),'Create Sub-Agent');
}

async function submitCreateSubAgent(parentId) {
    const modelSelect = document.getElementById('sub-agent-model');
    const modelCredentialId = modelSelect.value;

    if (!modelCredentialId) {
        showToast('Please select a model');
        return;
    }

    const selectedModel = subAgentModelOptions.find(m => m.value === modelCredentialId);
    const modelMatch = selectedModel?.label.match(/\(([^/]+)\/([^)]+)\)/);

    const params = {
        name: document.getElementById('sub-agent-name').value,
        description: document.getElementById('sub-agent-desc').value,
        instruction: document.getElementById('sub-agent-instr').value,
        globalInstruction: document.getElementById('sub-agent-ginstr').value,
        mode: document.getElementById('sub-agent-mode').value,
        modelName: modelMatch ? modelMatch[2] : '',
        modelCredentialId: modelCredentialId,
        credentialSource: 'app',
        parentAgentId: parentId,
    };

    try {
        await window.go.application.App.CreateSubAgent(params);
        hideModal();
        showAgentDetail(parentId);
    } catch (err) {
        showToast('Failed to create sub-agent: ' + err);
    }
}

async function showCreateAgentModal() {
    modelOptions = await window.go.application.App.GetModelOptions();

    const body = `
        <div class="form-group">
            <label for="agent-name">Name *</label>
            <input type="text" id="agent-name" placeholder="Agent name" required>
        </div>
        <div class="form-group">
            <label for="agent-desc">Description *</label>
            <textarea id="agent-desc" placeholder="Description" required></textarea>
        </div>
        <div class="form-group">
            <label for="agent-instr">Instructions *</label>
            <textarea id="agent-instr" placeholder="Instructions for the agent" required></textarea>
        </div>
        <div class="form-group">
            <label for="agent-ginstr">Global Instructions</label>
            <textarea id="agent-ginstr" placeholder="Global instructions of this agent. Will be used in all of it's sub-agents"></textarea>
        </div>
        <div class="form-group">
            <label for="agent-mode">Mode</label>
            <select id="agent-mode" required>
                <option value="chat">chat</option>
                <option value="task">task</option>
                <option value="single_turn">single_turn</option>
            </select>
        </div>
        <div class="form-group">
            <label for="agent-model">Model *</label>
            <select id="agent-model" required>
                <option value="">-- Select Model --</option>
                ${modelOptions.map(m => `<option value="${m.value}">${escapeHtml(m.label)}</option>`).join('')}
            </select>
        </div>
        <div class="accordion">
            <button type="button" class="accordion-toggle" onclick="toggleAccordion(this)">
                Model Configuration (Advanced) <span class="chevron">&#9654;</span>
            </button>
            <div class="accordion-content">
                <div class="form-group">
                    <label for="mc-temperature">Temperature (0.0 - 2.0)</label>
                    <input type="number" id="mc-temperature" step="0.1" min="0" max="2" placeholder="e.g. 0.7">
                </div>
                <div class="form-group">
                    <label for="mc-topP">Top P</label>
                    <input type="number" id="mc-topP" step="0.05" min="0" max="1" placeholder="e.g. 0.95">
                </div>
                <div class="form-group">
                    <label for="mc-topK">Top K</label>
                    <input type="number" id="mc-topK" min="1" placeholder="e.g. 40">
                </div>
                <div class="form-group">
                    <label for="mc-maxTokens">Max Output Tokens</label>
                    <input type="number" id="mc-maxTokens" min="1" placeholder="e.g. 1024">
                </div>
                <div class="form-group">
                    <label for="mc-frequencyPenalty">Frequency Penalty</label>
                    <input type="number" id="mc-frequencyPenalty" step="0.1" placeholder="e.g. 0.0">
                </div>
                <div class="form-group">
                    <label for="mc-presencePenalty">Presence Penalty</label>
                    <input type="number" id="mc-presencePenalty" step="0.1" placeholder="e.g. 0.0">
                </div>
                <div class="form-group">
                    <label for="mc-seed">Seed</label>
                    <input type="number" id="mc-seed" min="0" placeholder="e.g. 42">
                </div>
                <div class="form-group">
                    <label for="mc-stopSequences">Stop Sequences (comma separated)</label>
                    <input type="text" id="mc-stopSequences" placeholder="e.g. END, STOP">
                </div>
                <div class="form-group">
                    <label for="mc-responseMimeType">Response MIME Type</label>
                    <select id="mc-responseMimeType">
                        <option value="">-- Default --</option>
                        <option value="application/json">application/json</option>
                        <option value="text/plain">text/plain</option>
                    </select>
                </div>
                <div class="form-group">
                    <label for="mc-responseLogprobs">
                        <input type="checkbox" id="mc-responseLogprobs" style="width:auto;margin-right:6px;"> Return Log Probs
                    </label>
                </div>
                <div class="form-group">
                    <label for="mc-audioTimestamp">
                        <input type="checkbox" id="mc-audioTimestamp" style="width:auto;margin-right:6px;"> Audio Timestamp
                    </label>
                </div>
            </div>
        </div>`;

    showModal('Create Agent', body, submitCreateAgent,'Create Agent');
}

async function submitCreateAgent() {
    const modelSelect = document.getElementById('agent-model');
    const modelCredentialId = modelSelect.value;

    if (!modelCredentialId) {
        showToast('Please select a model');
        return;
    }

    const selectedModel = modelOptions.find(m => m.value === modelCredentialId);
    const modelMatch = selectedModel?.label.match(/\(([^/]+)\/([^)]+)\)/);

    const params = {
        name: document.getElementById('agent-name').value,
        description: document.getElementById('agent-desc').value,
        instruction: document.getElementById('agent-instr').value,
        globalInstruction: document.getElementById('agent-ginstr').value,
        mode: document.getElementById('agent-mode').value,
        modelName: modelMatch ? modelMatch[2] : '',
        modelCredentialId: modelCredentialId,
        credentialSource: 'app',
        modelConfig: buildModelConfig(),
    };

    try {
        await window.go.application.App.CreateAgent(params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to create agent: ' + err);
    }
}

function buildModelConfig() {
    const mc = {};
    const temp = document.getElementById('mc-temperature').value;
    const topP = document.getElementById('mc-topP').value;
    const topK = document.getElementById('mc-topK').value;
    const maxTokens = document.getElementById('mc-maxTokens').value;
    const freqPen = document.getElementById('mc-frequencyPenalty').value;
    const presPen = document.getElementById('mc-presencePenalty').value;
    const seed = document.getElementById('mc-seed').value;
    const stopSeqs = document.getElementById('mc-stopSequences').value;
    const mimeType = document.getElementById('mc-responseMimeType').value;
    const logprobs = document.getElementById('mc-responseLogprobs').checked;
    const audioTs = document.getElementById('mc-audioTimestamp').checked;

    if (temp !== '') mc.temperature = parseFloat(temp);
    if (topP !== '') mc.top_p = parseFloat(topP);
    if (topK !== '') mc.top_k = parseInt(topK);
    if (maxTokens !== '') mc.max_output_tokens = parseInt(maxTokens);
    if (freqPen !== '') mc.frequency_penalty = parseFloat(freqPen);
    if (presPen !== '') mc.presence_penalty = parseFloat(presPen);
    if (seed !== '') mc.seed = parseInt(seed);
    if (stopSeqs.trim() !== '') mc.stop_sequences = stopSeqs.split(',').map(s => s.trim()).filter(Boolean);
    if (mimeType !== '') mc.response_mime_type = mimeType;
    if (logprobs) mc.response_logprobs = true;
    if (audioTs) mc.audio_timestamp = true;

    return Object.keys(mc).length > 0 ? mc : null;
}

function toggleAccordion(btn) {
    btn.classList.toggle('open');
    const content = btn.nextElementSibling;
    content.classList.toggle('open');
}

// --- Agent navigation ---

function openRootAgentDetail(agentId) {
    agentNavStack = [agentId];
    showAgentDetail(agentId);
}

function openSubAgentDetail(agentId) {
    agentNavStack.push(agentId);
    showAgentDetail(agentId);
}

function goBackInAgentStack() {
    if (agentNavStack.length <= 1) {
        agentNavStack = [];
        renderAgents(document.getElementById('view-container'));
        return;
    }
    agentNavStack.pop();
    const prev = agentNavStack[agentNavStack.length - 1];
    showAgentDetail(prev);
}

// --- Sub-agent ordering ---

function renderSubAgentRow(sa, index, total) {
    // use array index + 1 (not sa.position): the backend stores position 0-based on create
    // but 1-based on reorder, so raw sa.position is not a consistent 1-based label.
    return `
        <div class="kb-item sub-agent-row" draggable="true" data-id="${sa.id}" data-index="${index}">
            <div class="sub-agent-order-controls">
                <span class="sub-agent-number">${index + 1}</span>
                <button class="sub-agent-arrow" title="Move up"
                    onclick="moveSubAgent('${sa.id}', -1)"
                    ${index === 0 ? 'disabled' : ''}>
                    &#9650;
                </button>
                <button class="sub-agent-arrow" title="Move down"
                    onclick="moveSubAgent('${sa.id}', 1)"
                    ${index === total - 1 ? 'disabled' : ''}>
                    &#9660;
                </button>
                <span class="sub-agent-drag-handle" title="Drag to reorder">&#9776;</span>
            </div>
            <span class="kb-item-title sub-agent-name" onclick="openSubAgentDetail('${sa.id}')">${escapeHtml(sa.name || sa.id)}</span>
        </div>`;
}

function renderSkillsSection(skills) {
    if (skills.length === 0) {
        return '<div class="empty-state" style="padding:20px">No skills attached</div>';
    }
    let html = '<div class="kb-list">';
    for (const s of skills) {
        const statusClass = s.isActive ? 'status-active' : 'status-inactive';
        const statusText = s.isActive ? 'Active' : 'Inactive';
        html += `
            <div class="kb-item">
                <span class="kb-item-title" onclick="showSkillDetail('${s.id}')" style="cursor:pointer;text-decoration:underline;text-decoration-color:transparent;transition:text-decoration-color 0.15s" onmouseenter="this.style.textDecorationColor='var(--accent)'" onmouseleave="this.style.textDecorationColor='transparent'">${escapeHtml(s.title || s.id)}</span>
                <div class="kb-item-actions">
                    <span class="card-subtitle">${escapeHtml(s.contentType)}</span>
                    <span class="status ${statusClass}">${statusText}</span>
                </div>
            </div>`;
    }
    html += '</div>';
    return html;
}

function renderMcpsSection(mcps) {
    if (mcps.length === 0) {
        return '<div class="empty-state" style="padding:20px">No MCP servers attached</div>';
    }
    let html = '<div class="kb-list">';
    for (const m of mcps) {
        const statusClass = m.isActive ? 'status-active' : 'status-inactive';
        const statusText = m.isActive ? 'Active' : 'Inactive';
        html += `
            <div class="kb-item">
                <span class="kb-item-title">${escapeHtml(m.name || m.id)}</span>
                <div class="kb-item-actions">
                    <span class="card-subtitle">${escapeHtml(m.transport)}</span>
                    <span class="card-subtitle" style="font-size:12px;opacity:0.7">${escapeHtml(m.endpoint)}</span>
                    <span class="status ${statusClass}">${statusText}</span>
                </div>
            </div>`;
    }
    html += '</div>';
    return html;
}

function rerenderSubAgentList() {
    const container = document.getElementById('sub-agents-container');
    if (!container) return;
    if (currentSubAgents.length === 0) {
        container.innerHTML = '<div class="empty-state" style="padding:20px">No sub-agents</div>';
        return;
    }
    let html = '<div class="kb-list">';
    for (let i = 0; i < currentSubAgents.length; i++) {
        html += renderSubAgentRow(currentSubAgents[i], i, currentSubAgents.length);
    }
    html += '</div>';
    container.innerHTML = html;
    attachSubAgentDragHandlers(container);
}

async function persistSubAgentOrder() {
    const orderIds = currentSubAgents.map(s => s.id);
    try {
        await window.go.application.App.ReorderSubAgents(currentAgentId, orderIds);
        const fresh = await window.go.application.App.GetSubAgents(currentAgentId);
        currentSubAgents = fresh;
        rerenderSubAgentList();
    } catch (err) {
        showToast('Failed to reorder sub-agents: ' + err);
    }
}

async function moveSubAgent(id, delta) {
    const idx = currentSubAgents.findIndex(s => s.id === id);
    const newIdx = idx + delta;
    if (idx < 0 || newIdx < 0 || newIdx >= currentSubAgents.length) return;
    const arr = currentSubAgents.slice();
    const [moved] = arr.splice(idx, 1);
    arr.splice(newIdx, 0, moved);
    currentSubAgents = arr;
    rerenderSubAgentList();
    await persistSubAgentOrder();
}

function attachSubAgentDragHandlers(scope) {
    const rows = scope.querySelectorAll('.sub-agent-row');
    rows.forEach(row => {
        row.addEventListener('dragstart', onDragStart);
        row.addEventListener('dragend', onDragEnd);
        row.addEventListener('dragover', onDragOver);
        row.addEventListener('drop', onDrop);
    });
}

function onDragStart(e) {
    e.dataTransfer.setData('text/plain', e.target.closest('.sub-agent-row').dataset.id);
    e.target.closest('.sub-agent-row').classList.add('dragging');
}

function onDragEnd(e) {
    e.target.closest('.sub-agent-row').classList.remove('dragging');
    document.querySelectorAll('.sub-agent-row.drop-target').forEach(el => el.classList.remove('drop-target'));
}

function onDragOver(e) {
    e.preventDefault();
    const target = e.target.closest('.sub-agent-row');
    if (!target) return;
    document.querySelectorAll('.sub-agent-row.drop-target').forEach(el => el.classList.remove('drop-target'));
    target.classList.add('drop-target');
}

function onDrop(e) {
    e.preventDefault();
    const targetRow = e.target.closest('.sub-agent-row');
    if (!targetRow) return;
    const fromId = e.dataTransfer.getData('text/plain');
    const toId = targetRow.dataset.id;
    const fromIdx = currentSubAgents.findIndex(s => s.id === fromId);
    const toIdx = currentSubAgents.findIndex(s => s.id === toId);
    if (fromIdx === -1 || toIdx === -1 || fromIdx === toIdx) return;
    const arr = currentSubAgents.slice();
    const [moved] = arr.splice(fromIdx, 1);
    arr.splice(toIdx, 0, moved);
    currentSubAgents = arr;
    rerenderSubAgentList();
    persistSubAgentOrder();
}

// --- MCP Servers ---

let mcpListData = [];

async function renderMCPs(container) {
    mcpListData = await window.go.application.App.GetMCPs();
    renderMCPList(container);
}

function renderMCPList(container) {
    let html = `
        <div class="page-header">
            <h1>MCP Servers</h1>
            <button class="btn btn-primary" onclick="showAddMCPModal()">Add MCP</button>
        </div>`;

    if (mcpListData.length === 0) {
        html += '<div class="empty-state">No MCP servers found</div>';
    } else {
        html += '<div class="cards-grid">';
        for (const m of mcpListData) {
            const statusClass = m.isActive ? 'status-active' : 'status-inactive';
            const statusText = m.isActive ? 'Active' : 'Inactive';

            html += `
                <div class="card clickable" onclick="showMCPDetail('${m.id}')">
                    <div class="card-title">${escapeHtml(m.name)}</div>
                    <div class="card-body">
                        <div class="card-label"><b>Transport:</b> ${escapeHtml(m.transport)}</div>
                        <div class="card-label"><b>Endpoint:</b> ${escapeHtml(m.endpoint)}</div>
                        ${m.command ? `<div class="card-label"><b>Command:</b> ${escapeHtml(m.command)}</div>` : ''}
                        <div><span class="status ${statusClass}">${statusText}</span></div>
                    </div>
                </div>`;
        }
        html += '</div>';
    }

    container.innerHTML = html;
}

async function showMCPDetail(mcpId) {
    const container = document.getElementById('view-container');
    let detail;
    try {
        detail = await window.go.application.App.GetMCP(mcpId);
    } catch (e) {
        showToast('Failed to load MCP details');
        return;
    }

    const statusClass = detail.isActive ? 'status-active' : 'status-inactive';
    const statusText = detail.isActive ? 'Active' : 'Inactive';
    const authType = detail.authType || 'none';

    let html = `
        <div class="page-header">
            <div class="page-header-left">
                <button class="skill-back-btn" onclick="renderMCPs(document.getElementById('view-container'))" title="Back to MCPs">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(detail.name)}</h1>
            </div>
            <div class="page-header-actions">
                <button class="btn btn-secondary btn-small" onclick="showEditMCPModal('${detail.id}')">Edit</button>
                <button class="btn btn-danger btn-small" onclick="confirmDeleteMCP('${detail.id}', '${escapeHtml(detail.name)}')">Delete</button>
            </div>
        </div>
        <div class="skill-detail-card">
            <div class="skill-detail-meta">
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Transport</span>
                    <span class="skill-detail-meta-value">${escapeHtml(detail.transport)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Status</span>
                    <span class="skill-detail-meta-value">
                        <span class="status ${statusClass}">${statusText}</span>
                    </span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Auth Type</span>
                    <span class="skill-detail-meta-value">${escapeHtml(authType)}</span>
                </div>
            </div>
            <div class="skill-detail-section">
                <div class="skill-detail-label">Endpoint</div>
                <pre class="skill-detail-content">${escapeHtml(detail.endpoint)}</pre>
            </div>
            ${detail.command ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Command</div>
                <pre class="skill-detail-content">${escapeHtml(detail.command)}</pre>
            </div>` : ''}
            ${detail.args && detail.args.length > 0 ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Arguments</div>
                <pre class="skill-detail-content">${escapeHtml(detail.args.join('\n'))}</pre>
            </div>` : ''}
            ${detail.allowedTools && detail.allowedTools.length > 0 ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Allowed Tools</div>
                <pre class="skill-detail-content">${escapeHtml(detail.allowedTools.join('\n'))}</pre>
            </div>` : ''}
            <div class="skill-detail-section">
                <div class="skill-detail-label">Require Confirmation</div>
                <pre class="skill-detail-content">${detail.requireConfirmation ? 'Yes' : 'No'}</pre>
            </div>
            ${authType === 'oauth' ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">OAuth Connection</div>
                <div class="skill-detail-content" id="mcp-oauth-status">Checking...</div>
            </div>` : ''}
        </div>`;

    container.innerHTML = html;

    if (authType === 'oauth') {
        renderOAuthStatus(detail.id);
    }
}

async function renderOAuthStatus(mcpId) {
    const statusEl = document.getElementById('mcp-oauth-status');
    if (!statusEl) return;
    let connected = false;
    try {
        connected = await window.go.application.App.IsOAuthConnected(mcpId);
    } catch (err) {
        statusEl.textContent = 'Unknown';
        return;
    }
    statusEl.innerHTML = `
        <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
            <span class="status ${connected ? 'status-active' : 'status-inactive'}">${connected ? 'Connected' : 'Not Connected'}</span>
            <button class="btn btn-small" onclick="connectOAuthMCP('${mcpId}')">${connected ? 'Reconnect' : 'Connect'}</button>
        </div>`;
}

async function connectOAuthMCP(mcpId) {
    try {
        showToast('Opening browser for authorization...');
        await window.go.application.App.ConnectOAuthMCP(mcpId);
        showToast('OAuth connected');
        renderOAuthStatus(mcpId);
        renderView();
    } catch (err) {
        showToast('OAuth connect failed: ' + err);
    }
}

// mcp is undefined for the create form and an mcpListData entry for edit.
function mcpFormHTML(mcp) {
    const m = mcp || {};
    const attr = (v, placeholder) => mcp ? `value="${escapeHtml(v)}"` : `placeholder="${placeholder}"`;
    const options = (pairs, current) => pairs
        .map(([v, label]) => `<option value="${v}" ${(current || '') === v ? 'selected' : ''}>${label}</option>`)
        .join('');
    const checkbox = (id, label, checked) => `
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="${id}" ${checked ? 'checked' : ''}> ${label}</label>
        </div>`;

    return `
        <div class="form-group">
            <label for="mcp-name">Name *</label>
            <input type="text" id="mcp-name" ${attr(m.name, 'MCP server name')} required>
        </div>
        <div class="form-group">
            <label for="mcp-transport">Transport *</label>
            <select id="mcp-transport">${options([['streamable_http', 'streamable_http'], ['stdio', 'stdio']], m.transport)}</select>
        </div>
        <div class="form-group">
            <label for="mcp-endpoint">Endpoint *</label>
            <input type="text" id="mcp-endpoint" ${attr(m.endpoint, 'e.g. http://localhost:3000/mcp')}>
        </div>
        <div class="form-group">
            <label for="mcp-command">Command</label>
            <input type="text" id="mcp-command" ${attr(m.command || '', 'e.g. npx')}>
        </div>
        <div class="form-group">
            <label for="mcp-args">Arguments (comma separated)</label>
            <input type="text" id="mcp-args" ${attr((m.args || []).join(', '), 'e.g. arg1, arg2')}>
        </div>
        <div class="form-group">
            <label for="mcp-auth-type">Auth Type</label>
            <select id="mcp-auth-type" onchange="toggleAuthConfig()">${options(
                [['', 'none'], ['api_key', 'api_key'], ['bearer', 'bearer'], ['oauth', 'oauth']], m.authType)}</select>
        </div>
        <div id="auth-config-group"></div>
        <div class="form-group">
            <label for="mcp-allowed-tools">Allowed Tools (comma separated)</label>
            <input type="text" id="mcp-allowed-tools" ${attr((m.allowedTools || []).join(', '), 'e.g. tool1, tool2')}>
        </div>
        ${checkbox('mcp-require-confirm', 'Require Confirmation', m.requireConfirmation)}
        ${checkbox('mcp-active', 'Active', mcp ? m.isActive : true)}`;
}

function showAddMCPModal() {
    showModal('Add MCP Server', mcpFormHTML(), submitMCPForm, 'Add');
    renderAuthConfigInputs({});
}

function showEditMCPModal(mcpId) {
    const mcp = mcpListData.find(m => m.id === mcpId);
    if (!mcp) return;
    showModal('Edit MCP Server', mcpFormHTML(mcp), () => submitMCPForm(mcpId), 'Update');
    renderAuthConfigInputs(mcp.authConfig || {});
}

async function submitMCPForm(mcpId) {
    const name = document.getElementById('mcp-name').value.trim();
    const endpoint = document.getElementById('mcp-endpoint').value.trim();
    const authTypeStr = document.getElementById('mcp-auth-type').value;

    if (!name || !endpoint) {
        showToast('Name and Endpoint are required');
        return;
    }

    const authConfig = buildAuthConfig();
    if (authTypeStr && !authConfig) {
        showToast('Auth Config is required for the selected Auth Type');
        return;
    }

    const csv = (s) => s ? s.split(',').map(v => v.trim()).filter(Boolean) : [];
    const params = {
        name,
        transport: document.getElementById('mcp-transport').value,
        endpoint,
        command: document.getElementById('mcp-command').value.trim() || '',
        args: csv(document.getElementById('mcp-args').value.trim()),
        authType: authTypeStr || null,
        authConfig,
        allowedTools: csv(document.getElementById('mcp-allowed-tools').value.trim()),
        requireConfirmation: document.getElementById('mcp-require-confirm').checked,
        isActive: document.getElementById('mcp-active').checked,
    };

    try {
        if (mcpId) {
            await window.go.application.App.UpdateMCP(mcpId, params);
        } else {
            await window.go.application.App.CreateMCP(params);
        }
        hideModal();
        renderView();
    } catch (err) {
        showToast(`Failed to ${mcpId ? 'update' : 'add'} MCP: ` + err);
    }
}

function confirmDeleteMCP(mcpId, mcpName) {
    const body = `<p>Are you sure you want to delete the MCP server "${escapeHtml(mcpName)}"?</p>`;
    showModal('Delete MCP Server', body, () => deleteMCP(mcpId), 'Delete');
}

async function deleteMCP(mcpId) {
    try {
        await window.go.application.App.DeleteMCP(mcpId);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to delete MCP: ' + err);
    }
}

// --- MCP Auth Config ---

let mcpAuthConfigState = {};

function collectAuthConfigInputs() {
    const cfg = Object.assign({}, mcpAuthConfigState);
    document.querySelectorAll('[data-auth-field]').forEach(el => {
        cfg[el.dataset.authField] = el.value;
    });
    return cfg;
}

function toggleAuthConfig() {
    mcpAuthConfigState = collectAuthConfigInputs();
    renderAuthConfigInputs();
}

function renderAuthConfigInputs(initial) {
    if (initial) {
        mcpAuthConfigState = Object.assign({}, initial);
    }
    const type = document.getElementById('mcp-auth-type').value;
    const group = document.getElementById('auth-config-group');
    const c = mcpAuthConfigState;
    let html = '';

    if (type === 'bearer') {
        html = `
            <div class="form-group">
                <label for="mcp-auth-bearer">Bearer Token *</label>
                <input type="password" id="mcp-auth-bearer" data-auth-field="bearer" placeholder="Bearer token" value="${escapeHtml(c.bearer || '')}">
            </div>`;
    } else if (type === 'api_key') {
        html = `
            <div class="form-group">
                <label for="mcp-auth-name">Header Name *</label>
                <input type="text" id="mcp-auth-name" data-auth-field="name" placeholder="X-Api-Key" value="${escapeHtml(c.name || 'X-Api-Key')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-value">API Key Value *</label>
                <input type="password" id="mcp-auth-value" data-auth-field="value" placeholder="API key value">
            </div>`;
    } else if (type === 'oauth') {
        html = `
            <div class="form-group">
                <label for="mcp-auth-client-id">Client ID *</label>
                <input type="text" id="mcp-auth-client-id" data-auth-field="client_id" placeholder="OAuth client id" value="${escapeHtml(c.client_id || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-client-secret">Client Secret</label>
                <input type="password" id="mcp-auth-client-secret" data-auth-field="client_secret" placeholder="Optional for public/PKCE-only clients" value="${escapeHtml(c.client_secret || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-scopes">Scopes (comma separated)</label>
                <input type="text" id="mcp-auth-scopes" data-auth-field="scopes" placeholder="e.g. openid, profile, email" value="${escapeHtml(c.scopes || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-auth-url">Authorization URL (auto-discovered if empty)</label>
                <input type="text" id="mcp-auth-auth-url" data-auth-field="auth_url" placeholder="https://provider.example.com/authorize" value="${escapeHtml(c.auth_url || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-token-url">Token URL (auto-discovered if empty)</label>
                <input type="text" id="mcp-auth-token-url" data-auth-field="token_url" placeholder="https://provider.example.com/token" value="${escapeHtml(c.token_url || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-redirect-uri">Redirect URI</label>
                <input type="text" id="mcp-auth-redirect-uri" data-auth-field="redirect_uri" placeholder="http://127.0.0.1:48421/callback" value="${escapeHtml(c.redirect_uri || '')}">
            </div>
            <div class="form-group">
                <label for="mcp-auth-style">Auth Style</label>
                <select id="mcp-auth-style" data-auth-field="auth_style">
                    <option value="auto" ${(c.auth_style || 'auto') === 'auto' ? 'selected' : ''}>auto</option>
                    <option value="in_header" ${c.auth_style === 'in_header' ? 'selected' : ''}>in_header</option>
                    <option value="in_params" ${c.auth_style === 'in_params' ? 'selected' : ''}>in_params</option>
                </select>
            </div>`;
    }

    group.innerHTML = html;
}

function buildAuthConfig() {
    const type = document.getElementById('mcp-auth-type').value;
    const c = collectAuthConfigInputs();
    if (!type) return null;

    if (type === 'bearer') {
        const bearer = (c.bearer || '').trim();
        return bearer ? {bearer} : null;
    }
    if (type === 'api_key') {
        const name = (c.name || 'X-Api-Key').trim();
        const value = (c.value || '').trim();
        return name && value ? {name, value} : null;
    }
    if (type === 'oauth') {
        const clientId = (c.client_id || '').trim();
        if (!clientId) return null;
        const scopes = (c.scopes || '').split(',').map(s => s.trim()).filter(Boolean);
        const cfg = {
            client_id: clientId,
            client_secret: (c.client_secret || '').trim(),
            scopes: scopes,
            auth_url: (c.auth_url || '').trim(),
            token_url: (c.token_url || '').trim(),
            redirect_uri: (c.redirect_uri || '').trim(),
            auth_style: c.auth_style || 'auto',
        };
        for (const k of Object.keys(cfg)) {
            if (cfg[k] === '' || (Array.isArray(cfg[k]) && cfg[k].length === 0)) delete cfg[k];
        }
        return cfg;
    }
    return null;
}

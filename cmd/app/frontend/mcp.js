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
                <div class="card" onclick="showMCPDetail('${m.id}')" style="cursor:pointer">
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
            <div style="display:flex;align-items:center;gap:8px">
                <button class="skill-back-btn" onclick="renderMCPs(document.getElementById('view-container'))" title="Back to MCPs">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(detail.name)}</h1>
            </div>
            <div style="display:flex;gap:8px">
                <button class="btn btn-small" onclick="showEditMCPModal('${detail.id}')">Edit</button>
                <button class="btn btn-small" style="background:#e74c3c;color:#fff" onclick="confirmDeleteMCP('${detail.id}', '${escapeHtml(detail.name)}')">Delete</button>
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
        </div>`;

    container.innerHTML = html;
}

function showAddMCPModal() {
    const body = `
        <div class="form-group">
            <label for="mcp-name">Name *</label>
            <input type="text" id="mcp-name" placeholder="MCP server name" required>
        </div>
        <div class="form-group">
            <label for="mcp-transport">Transport *</label>
            <select id="mcp-transport">
                <option value="streamable_http">streamable_http</option>
                <option value="stdio">stdio</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mcp-endpoint">Endpoint *</label>
            <input type="text" id="mcp-endpoint" placeholder="e.g. http://localhost:3000/mcp">
        </div>
        <div class="form-group">
            <label for="mcp-command">Command</label>
            <input type="text" id="mcp-command" placeholder="e.g. npx">
        </div>
        <div class="form-group">
            <label for="mcp-args">Arguments (comma separated)</label>
            <input type="text" id="mcp-args" placeholder="e.g. arg1, arg2">
        </div>
        <div class="form-group">
            <label for="mcp-auth-type">Auth Type</label>
            <select id="mcp-auth-type">
                <option value="">none</option>
                <option value="api_key">api_key</option>
                <option value="oauth">oauth</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mcp-allowed-tools">Allowed Tools (comma separated)</label>
            <input type="text" id="mcp-allowed-tools" placeholder="e.g. tool1, tool2">
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mcp-require-confirm"> Require Confirmation</label>
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mcp-active" checked> Active</label>
        </div>`;

    showModal('Add MCP Server', body, submitAddMCP, 'Add');
}

async function submitAddMCP() {
    const name = document.getElementById('mcp-name').value.trim();
    const transport = document.getElementById('mcp-transport').value;
    const endpoint = document.getElementById('mcp-endpoint').value.trim();
    const commandStr = document.getElementById('mcp-command').value.trim();
    const argsStr = document.getElementById('mcp-args').value.trim();
    const authTypeStr = document.getElementById('mcp-auth-type').value;
    const allowedToolsStr = document.getElementById('mcp-allowed-tools').value.trim();
    const requireConfirm = document.getElementById('mcp-require-confirm').checked;
    const isActive = document.getElementById('mcp-active').checked;

    if (!name || !endpoint) {
        showToast('Name and Endpoint are required');
        return;
    }

    const params = {
        name: name,
        transport: transport,
        endpoint: endpoint,
        command: commandStr || '',
        args: argsStr ? argsStr.split(',').map(s => s.trim()).filter(Boolean) : [],
        authType: authTypeStr || null,
        allowedTools: allowedToolsStr ? allowedToolsStr.split(',').map(s => s.trim()).filter(Boolean) : [],
        requireConfirmation: requireConfirm,
        isActive: isActive,
    };

    try {
        await window.go.application.App.CreateMCP(params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to add MCP: ' + err);
    }
}

function showEditMCPModal(mcpId) {
    const mcp = mcpListData.find(m => m.id === mcpId);
    if (!mcp) return;

    const body = `
        <div class="form-group">
            <label for="mcp-name">Name *</label>
            <input type="text" id="mcp-name" value="${escapeHtml(mcp.name)}" required>
        </div>
        <div class="form-group">
            <label for="mcp-transport">Transport *</label>
            <select id="mcp-transport">
                <option value="streamable_http" ${mcp.transport === 'streamable_http' ? 'selected' : ''}>streamable_http</option>
                <option value="stdio" ${mcp.transport === 'stdio' ? 'selected' : ''}>stdio</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mcp-endpoint">Endpoint *</label>
            <input type="text" id="mcp-endpoint" value="${escapeHtml(mcp.endpoint)}">
        </div>
        <div class="form-group">
            <label for="mcp-command">Command</label>
            <input type="text" id="mcp-command" value="${escapeHtml(mcp.command || '')}">
        </div>
        <div class="form-group">
            <label for="mcp-args">Arguments (comma separated)</label>
            <input type="text" id="mcp-args" value="${escapeHtml((mcp.args || []).join(', '))}">
        </div>
        <div class="form-group">
            <label for="mcp-auth-type">Auth Type</label>
            <select id="mcp-auth-type">
                <option value="" ${(mcp.authType || '') === '' ? 'selected' : ''}>none</option>
                <option value="api_key" ${mcp.authType === 'api_key' ? 'selected' : ''}>api_key</option>
                <option value="oauth" ${mcp.authType === 'oauth' ? 'selected' : ''}>oauth</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mcp-allowed-tools">Allowed Tools (comma separated)</label>
            <input type="text" id="mcp-allowed-tools" value="${escapeHtml((mcp.allowedTools || []).join(', '))}">
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mcp-require-confirm" ${mcp.requireConfirmation ? 'checked' : ''}> Require Confirmation</label>
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mcp-active" ${mcp.isActive ? 'checked' : ''}> Active</label>
        </div>`;

    showModal('Edit MCP Server', body, () => submitEditMCP(mcpId), 'Update');
}

async function submitEditMCP(mcpId) {
    const name = document.getElementById('mcp-name').value.trim();
    const transport = document.getElementById('mcp-transport').value;
    const endpoint = document.getElementById('mcp-endpoint').value.trim();
    const commandStr = document.getElementById('mcp-command').value.trim();
    const argsStr = document.getElementById('mcp-args').value.trim();
    const authTypeStr = document.getElementById('mcp-auth-type').value;
    const allowedToolsStr = document.getElementById('mcp-allowed-tools').value.trim();
    const requireConfirm = document.getElementById('mcp-require-confirm').checked;
    const isActive = document.getElementById('mcp-active').checked;

    if (!name || !endpoint) {
        showToast('Name and Endpoint are required');
        return;
    }

    const params = {
        name: name,
        transport: transport,
        endpoint: endpoint,
        command: commandStr || '',
        args: argsStr ? argsStr.split(',').map(s => s.trim()).filter(Boolean) : [],
        authType: authTypeStr || null,
        allowedTools: allowedToolsStr ? allowedToolsStr.split(',').map(s => s.trim()).filter(Boolean) : [],
        requireConfirmation: requireConfirm,
        isActive: isActive,
    };

    try {
        await window.go.application.App.UpdateMCP(mcpId, params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to update MCP: ' + err);
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

// --- Model Credentials ---

let modelListData = [];

async function renderModels(container) {
    modelListData = await window.go.application.App.GetModelCredentials();

    renderModelList(container);
}

function renderModelList(container) {
    let html = `
        <div class="page-header">
            <h1>Models</h1>
            <button class="btn btn-primary" onclick="showCreateModelModal()">Add Model Credential</button>
        </div>`;

    if (modelListData.length === 0) {
        html += '<div class="empty-state">No model credentials found</div>';
    } else {
        html += '<div class="cards-grid models-grid">';
        for (const c of modelListData) {
            const statusClass = c.isActive ? 'status-active' : 'status-inactive';
            const statusText = c.isActive ? 'Active' : 'Inactive';

            html += `
                <div class="card" onclick="showModelDetail('${c.id}')" style="cursor:pointer">
                    <div class="card-title">${escapeHtml(c.title)}</div>
                    <div class="card-body">
                        <div class="card-label"><b>Provider:</b> ${escapeHtml(c.provider)}</div>
                        <div class="card-label"><b>Model:</b> <strong>${escapeHtml(c.modelName)}</strong></div>
                        ${c.baseUrl ? `<div class="card-label"><b>Endpoint:</b> ${escapeHtml(c.baseUrl)}</div>` : ''}
                        <div><span class="status ${statusClass}">${statusText}</span></div>
                    </div>
                </div>`;
        }
        html += '</div>';
    }

    container.innerHTML = html;
}

async function showModelDetail(modelId) {
    const container = document.getElementById('view-container');
    const model = modelListData.find(i => i.id === modelId);
    if (!model) return;

    const statusClass = model.isActive ? 'status-active' : 'status-inactive';
    const statusText = model.isActive ? 'Active' : 'Inactive';

    let html = `
        <div class="page-header">
            <div style="display:flex;align-items:center;gap:8px">
                <button class="skill-back-btn" onclick="renderModels(document.getElementById('view-container'))" title="Back to models">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(model.title)}</h1>
            </div>
            <div style="display:flex;gap:8px">
                <button class="btn btn-small" onclick="showEditModelModal('${model.id}')">Edit</button>
                <button class="btn btn-small" style="background:#e74c3c;color:#fff" onclick="confirmDeleteModel('${model.id}', '${escapeHtml(model.title)}')">Delete</button>
            </div>
        </div>
        <div class="skill-detail-card">
            <div class="skill-detail-meta">
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Provider</span>
                    <span class="skill-detail-meta-value">${escapeHtml(model.provider)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Model</span>
                    <span class="skill-detail-meta-value">${escapeHtml(model.modelName)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Status</span>
                    <span class="skill-detail-meta-value">
                        <span class="status ${statusClass}">${statusText}</span>
                    </span>
                </div>
            </div>
            ${model.baseUrl ? `
            <div class="skill-detail-section">
                <div class="skill-detail-label">Endpoint</div>
                <pre class="skill-detail-content">${escapeHtml(model.baseUrl)}</pre>
            </div>` : ''}
        </div>`;

    container.innerHTML = html;
}

function confirmDeleteModel(modelId, modelTitle) {
    const body = `<p>Are you sure you want to delete the model "${escapeHtml(modelTitle)}"?</p>`;
    showModal('Delete Model', body, () => deleteModel(modelId), 'Delete Model');
}

async function deleteModel(modelId) {
    try {
        await window.go.application.App.DeleteModel(modelId);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to delete model: ' + err);
    }
}

function showCreateModelModal() {
    const body = `
        <div class="form-group">
            <label for="mc-title">Title *</label>
            <input type="text" id="mc-title" maxlength="20" placeholder="Short title (max 20 chars)">
        </div>
        <div class="form-group">
            <label for="mc-provider">Provider *</label>
            <select id="mc-provider" onchange="onProviderChange()">
                <option value="openai">openai</option>
                <option value="openrouter">openrouter</option>
                <option value="anthropic">anthropic</option>
                <option value="google">google</option>
                <option value="groq">groq</option>
                <option value="deepseek">deepseek</option>
                <option value="minimax">minimax</option>
                <option value="ollama">ollama</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mc-model">Model Name *</label>
            <input type="text" id="mc-model" placeholder="e.g. gpt-4o">
        </div>
        <div class="form-group">
            <label for="mc-url">Base URL</label>
            <input type="text" id="mc-url" value="https://api.openai.com/v1">
        </div>
        <div class="form-group">
            <label for="mc-key">API Key</label>
            <input type="password" id="mc-key" placeholder="API key">
        </div>`;

    showModal('Add Model Credential', body, submitCreateModel, 'Add Model');
}

function showEditModelModal(modelId) {
    const model = modelListData.find(i => i.id === modelId);
    if (!model) return;

    const body = `
        <div class="form-group">
            <label for="mc-title">Title *</label>
            <input type="text" id="mc-title" maxlength="20" value="${escapeHtml(model.title)}" placeholder="Short title (max 20 chars)">
        </div>
        <div class="form-group">
            <label for="mc-provider">Provider *</label>
            <select id="mc-provider" onchange="onProviderChange()">
                <option value="openai" ${model.provider === 'openai' ? 'selected' : ''}>openai</option>
                <option value="openrouter" ${model.provider === 'openrouter' ? 'selected' : ''}>openrouter</option>
                <option value="anthropic" ${model.provider === 'anthropic' ? 'selected' : ''}>anthropic</option>
                <option value="google" ${model.provider === 'google' ? 'selected' : ''}>google</option>
                <option value="groq" ${model.provider === 'groq' ? 'selected' : ''}>groq</option>
                <option value="deepseek" ${model.provider === 'deepseek' ? 'selected' : ''}>deepseek</option>
                <option value="minimax" ${model.provider === 'minimax' ? 'selected' : ''}>minimax</option>
                <option value="ollama" ${model.provider === 'ollama' ? 'selected' : ''}>ollama</option>
            </select>
        </div>
        <div class="form-group">
            <label for="mc-model">Model Name *</label>
            <input type="text" id="mc-model" value="${escapeHtml(model.modelName)}" placeholder="e.g. gpt-4o">
        </div>
        <div class="form-group">
            <label for="mc-url">Base URL</label>
            <input type="text" id="mc-url" value="${escapeHtml(model.baseUrl || '')}">
        </div>
        <div class="form-group">
            <label for="mc-key">API Key</label>
            <input type="password" id="mc-key" placeholder="Leave blank to keep current key">
        </div>
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mc-active" ${model.isActive ? 'checked' : ''}> Active</label>
        </div>`;

    showModal('Edit Model Credential', body, () => submitEditModel(modelId),'Update');
}

async function submitEditModel(modelId) {
    const params = {
        title: document.getElementById('mc-title').value,
        provider: document.getElementById('mc-provider').value,
        modelName: document.getElementById('mc-model').value,
        baseUrl: document.getElementById('mc-url').value,
        apiKey: document.getElementById('mc-key').value,
        isActive: document.getElementById('mc-active').checked,
    };

    if (!params.title.trim()) {
        showToast('Title is required');
        return;
    }

    try {
        await window.go.application.App.UpdateModelCredential(modelId, params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to update model credential: ' + err);
    }
}

const providerURLs = {
    openai: 'https://api.openai.com/v1',
    openrouter: 'https://openrouter.ai/api/v1',
    anthropic: 'https://api.anthropic.com/v1',
    google: 'https://generativelanguage.googleapis.com/v1beta',
    groq: 'https://api.groq.com/openai/v1',
    deepseek: 'https://api.deepseek.com/v1',
    minimax: 'https://api.minimax.chat/v1',
    ollama: 'http://localhost:11434/v1',
};

function onProviderChange() {
    const provider = document.getElementById('mc-provider').value;
    const urlInput = document.getElementById('mc-url');
    if (providerURLs[provider]) {
        urlInput.value = providerURLs[provider];
    }
}

async function submitCreateModel() {
    const params = {
        title: document.getElementById('mc-title').value,
        provider: document.getElementById('mc-provider').value,
        modelName: document.getElementById('mc-model').value,
        baseUrl: document.getElementById('mc-url').value,
        apiKey: document.getElementById('mc-key').value,
        scope: 'app',
    };

    if (!params.title.trim()) {
        showToast('Title is required');
        return;
    }

    try {
        await window.go.application.App.CreateModelCredential(params);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to create model credential: ' + err);
    }
}

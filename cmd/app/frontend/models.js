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
                <div class="card clickable" onclick="showModelDetail('${c.id}')">
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
            <div class="page-header-left">
                <button class="skill-back-btn" onclick="renderModels(document.getElementById('view-container'))" title="Back to models">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(model.title)}</h1>
            </div>
            <div class="page-header-actions">
                <button class="btn btn-secondary btn-small" onclick="showEditModelModal('${model.id}')">Edit</button>
                <button class="btn btn-danger btn-small" onclick="confirmDeleteModel('${model.id}', '${escapeHtml(model.title)}')">Delete</button>
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

const MODEL_PROVIDERS = ['openai', 'openrouter', 'anthropic', 'google', 'groq', 'deepseek', 'minimax', 'ollama'];

// model is undefined for the create form and a modelListData entry for edit.
function modelFormHTML(model) {
    const m = model || {};
    const selected = model ? (model.baseUrl || '') : 'https://api.openai.com/v1';
    return `
        <div class="form-group">
            <label for="mc-title">Title *</label>
            <input type="text" id="mc-title" maxlength="20" value="${escapeHtml(m.title)}" placeholder="Short title (max 20 chars)">
        </div>
        <div class="form-group">
            <label for="mc-provider">Provider *</label>
            <select id="mc-provider" onchange="onProviderChange()">
                ${MODEL_PROVIDERS.map(p => `<option value="${p}" ${m.provider === p ? 'selected' : ''}>${p}</option>`).join('')}
            </select>
        </div>
        <div class="form-group">
            <label for="mc-model">Model Name *</label>
            <input type="text" id="mc-model" value="${escapeHtml(m.modelName)}" placeholder="e.g. gpt-4o">
        </div>
        <div class="form-group">
            <label for="mc-url">Base URL</label>
            <input type="text" id="mc-url" value="${escapeHtml(selected)}">
        </div>
        <div class="form-group">
            <label for="mc-key">API Key</label>
            <input type="password" id="mc-key" placeholder="${model ? 'Leave blank to keep current key' : 'API key'}">
        </div>
        ${model ? `
        <div class="form-group">
            <label class="checkbox-label"><input type="checkbox" id="mc-active" ${m.isActive ? 'checked' : ''}> Active</label>
        </div>` : ''}`;
}

function showCreateModelModal() {
    showModal('Add Model Credential', modelFormHTML(), submitModelForm, 'Add Model');
}

function showEditModelModal(modelId) {
    const model = modelListData.find(i => i.id === modelId);
    if (!model) return;
    showModal('Edit Model Credential', modelFormHTML(model), () => submitModelForm(modelId), 'Update');
}

async function submitModelForm(modelId) {
    const params = {
        title: document.getElementById('mc-title').value,
        provider: document.getElementById('mc-provider').value,
        modelName: document.getElementById('mc-model').value,
        baseUrl: document.getElementById('mc-url').value,
        apiKey: document.getElementById('mc-key').value,
        ...(modelId
            ? { isActive: document.getElementById('mc-active').checked }
            : { scope: 'app' }),
    };

    if (!params.title.trim()) {
        showToast('Title is required');
        return;
    }

    try {
        if (modelId) {
            await window.go.application.App.UpdateModelCredential(modelId, params);
        } else {
            await window.go.application.App.CreateModelCredential(params);
        }
        hideModal();
        renderView();
    } catch (err) {
        showToast(`Failed to ${modelId ? 'update' : 'create'} model credential: ` + err);
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

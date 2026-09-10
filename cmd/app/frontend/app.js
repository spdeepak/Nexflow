let currentView = 'agents';

const VIEW_STORAGE_KEY = 'nexflow.currentView';

// --- View Switching ---

function switchView(view) {
    currentView = view;
    localStorage.setItem(VIEW_STORAGE_KEY, view);
    document.querySelectorAll('.sidebar-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === view);
    });
    renderView();
}

async function renderView() {
    const container = document.getElementById('view-container');
    container.innerHTML = '<div class="loading">Loading...</div>';

    try {
        switch (currentView) {
            case 'agents':
                await renderAgents(container);
                break;
            case 'skill':
                await renderSkill(container);
                break;
            case 'models':
                await renderModels(container);
                break;
            case 'mcp':
                await renderMCPs(container);
                break;
            case 'chat':
                await renderChat(container);
                break;
        }
    } catch (err) {
        container.innerHTML = `<div class="empty-state">Error: ${escapeHtml(err)}</div>`;
    }
}

// --- Modal Helpers ---

function showModal(title, bodyHTML, onSubmit, submitLabel = 'Create') {
    document.getElementById('modal-title').textContent = title;
    document.getElementById('modal-body').innerHTML = `
        <form id="modal-form">
            ${bodyHTML}
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary" id="modal-submit">${submitLabel}</button>
            </div>
        </form>`;

    const form = document.getElementById('modal-form');
    form._onSubmit = onSubmit;
    form.addEventListener('submit', (e) => {
        e.preventDefault();
        if (typeof form._onSubmit === 'function') {
            form._onSubmit();
        }
    });
    document.getElementById('modal-overlay').classList.remove('hidden');
}

function hideModal() {
    document.getElementById('modal-overlay').classList.add('hidden');
}

function closeModal(e) {
    if (e.target === document.getElementById('modal-overlay')) {
        hideModal();
    }
}

// --- Toast ---

function showToast(msg) {
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
}

// --- Helpers ---

function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// --- Sidebar user ---

async function renderUser() {
    try {
        const name = await window.go.application.App.GetUsername();
        document.getElementById('user-name').textContent = name || 'User';
        const initial = (name || 'U').charAt(0);
        document.getElementById('user-avatar').textContent = initial;
    } catch (err) {
        document.getElementById('user-name').textContent = 'Unknown';
        document.getElementById('user-avatar').textContent = '?';
    }
}

// --- Init ---

window.addEventListener('DOMContentLoaded', () => {
    renderUser();

    const saved = localStorage.getItem(VIEW_STORAGE_KEY);
    if (saved && ['agents', 'skill', 'models', 'chat', 'mcp'].includes(saved)) {
        currentView = saved;
    }
    switchView(currentView);
});

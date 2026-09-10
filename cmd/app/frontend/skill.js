// --- Skill ---

let skillListData = [];

async function renderSkill(container) {
    skillListData = await window.go.application.App.GetSkill();

    renderSkillList(container);
}

function renderSkillList(container) {
    let html = `
        <div class="page-header">
            <h1>Skills</h1>
            <button class="btn btn-primary" onclick="showCreateSkillModal()">Create Skill</button>
        </div>`;

    if (skillListData.length === 0) {
        html += '<div class="empty-state">No skill items found</div>';
    } else {
        html += '<div class="kb-list">';
        for (const k of skillListData) {
            html += `<div class="kb-item" data-id="${k.id}" data-title="${escapeHtml(k.title)}">
                ${iconSvg('bullet', 'skill-bullet')}
                <span class="kb-item-title" onclick="showSkillDetail('${k.id}')" style="cursor:pointer;text-decoration:underline;text-decoration-color:transparent;transition:text-decoration-color 0.15s" onmouseenter="this.style.textDecorationColor='var(--accent)'" onmouseleave="this.style.textDecorationColor='transparent'">${escapeHtml(k.title)}</span>
                <div class="kb-item-actions">
                    <button class="btn btn-secondary btn-small" onclick="showEditSkillModal(skillListData.find(i => i.id === '${k.id}'))">Edit</button>
                    <button class="btn btn-small" style="background:#e74c3c;color:#fff" onclick="confirmDeleteSkill('${k.id}', '${escapeHtml(k.title)}')">Delete</button>
                </div>
            </div>`;
        }
        html += '</div>';
    }

    container.innerHTML = html;
}

function switchSkillTab(tabBtn, panelId) {
    tabBtn.closest('.skill-detail-card').querySelectorAll('.skill-tab').forEach(t => t.classList.remove('active'));
    tabBtn.closest('.skill-detail-card').querySelectorAll('.skill-tab-panel').forEach(p => p.classList.remove('active'));
    tabBtn.classList.add('active');
    document.getElementById(panelId).classList.add('active');
}

async function showSkillDetail(skillId) {
    const container = document.getElementById('view-container');
    const skill = skillListData.find(i => i.id === skillId);
    if (!skill) return;

    let contentSection = '';
    if (skill.contentType === 'pdf') {
        let pdfEmbed = '<p style="color:var(--text-muted)">No PDF file</p>';
        try {
            const base64 = await window.go.application.App.GetSkillPDF(skill.id);
            if (base64) {
                const dataUri = `data:application/pdf;base64,${base64}`;
                pdfEmbed = `<embed src="${dataUri}" type="application/pdf" width="100%" height="500px" style="border: 1px solid var(--border); border-radius: 8px;">`;
            }
        } catch (e) {
            console.error('Failed to load PDF:', e);
            pdfEmbed = '<p style="color:var(--text-muted)">Failed to load PDF</p>';
        }
        contentSection = `
        <div class="skill-detail-tabs">
            <button class="skill-tab active" onclick="switchSkillTab(this, 'skill-tab-pdf')">PDF</button>
            <button class="skill-tab" onclick="switchSkillTab(this, 'skill-tab-content')">Content</button>
        </div>
        <div class="skill-tab-panel active" id="skill-tab-pdf">
            ${pdfEmbed}
        </div>
        <div class="skill-tab-panel" id="skill-tab-content">
            <pre class="skill-detail-content">${escapeHtml(skill.content || '')}</pre>
        </div>`;
    } else {
        contentSection = `
        <div class="skill-detail-section">
            <div class="skill-detail-label">Content</div>
            <pre class="skill-detail-content">${escapeHtml(skill.content || '')}</pre>
        </div>`;
    }

    let html = `
        <div class="page-header">
            <div style="display:flex;align-items:center;gap:8px">
                <button class="skill-back-btn" onclick="renderSkill(document.getElementById('view-container'))" title="Back to skills">
                    ${iconSvg('back', 'icon back-icon')}
                </button>
                <h1>${escapeHtml(skill.title)}</h1>
            </div>
            <div style="display:flex;gap:8px">
                <button class="btn btn-secondary btn-small" onclick="showEditSkillModal(skillListData.find(i => i.id === '${skill.id}'))">Edit</button>
                <button class="btn btn-small" style="background:#e74c3c;color:#fff" onclick="confirmDeleteSkill('${skill.id}', '${escapeHtml(skill.title)}')">Delete</button>
            </div>
        </div>
        <div class="skill-detail-card">
            <div class="skill-detail-meta">
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Type</span>
                    <span class="skill-detail-meta-value">${escapeHtml(skill.contentType)}</span>
                </div>
                <div class="skill-detail-meta-item">
                    <span class="skill-detail-meta-label">Status</span>
                    <span class="skill-detail-meta-value">
                        <span class="status ${skill.isActive ? 'status-active' : 'status-inactive'}">${skill.isActive ? 'Active' : 'Inactive'}</span>
                    </span>
                </div>
            </div>
            ${contentSection}
        </div>`;

    container.innerHTML = html;
}

function showCreateSkillModal() {
    const body = `
        <div class="form-group">
            <label for="kb-title">Title *</label>
            <input type="text" id="kb-title" placeholder="Title (required)" required>
        </div>
        <div class="form-group">
            <label for="kb-type">Content Type</label>
            <select id="kb-type">
                <option value="text">text</option>
<!--                <option value="markdown">markdown</option>-->
                <option value="pdf">pdf</option>
<!--                <option value="csv">csv</option>-->
<!--                <option value="json">json</option>-->
            </select>
        </div>
        <div class="form-group" id="kb-file-group">
            <label for="kb-file">Upload PDF</label>
            <input type="file" id="kb-file" accept=".pdf">
        </div>
        <div class="form-group" id="kb-content-group">
            <label for="kb-content">Skill</label>
            <textarea id="kb-content" placeholder="Skill"></textarea>
        </div>
        <div class="form-group">
            <label for="kb-scope">Scope</label>
            <select id="kb-scope">
                <option value="user">user</option>
                <option value="app">app</option>
            </select>
        </div>`;

    showModal('Create Skill', body, submitCreateSkill);

    const typeSelect = document.getElementById('kb-type');
    const fileGroup = document.getElementById('kb-file-group');
    const contentGroup = document.getElementById('kb-content-group');

    function toggleInputs() {
        if (typeSelect.value === 'pdf') {
            fileGroup.style.display = '';
            contentGroup.style.display = 'none';
        } else {
            fileGroup.style.display = 'none';
            contentGroup.style.display = '';
        }
    }

    typeSelect.addEventListener('change', toggleInputs);
    toggleInputs();
}

async function submitCreateSkill() {
    const contentType = document.getElementById('kb-type').value;

    const params = {
        title: document.getElementById('kb-title').value,
        contentType: contentType,
        scope: document.getElementById('kb-scope').value,
    };

    try {
        if (contentType === 'pdf') {
            const fileInput = document.getElementById('kb-file');
            const file = fileInput.files[0];
            if (!file) {
                showToast('Please select a PDF file');
                return;
            }

            const arrayBuffer = await file.arrayBuffer();
            const bytes = new Uint8Array(arrayBuffer);
            let binary = '';
            for (let i = 0; i < bytes.byteLength; i++) {
                binary += String.fromCharCode(bytes[i]);
            }
            const fileDataBase64 = btoa(binary);

            await window.go.application.App.CreateSkillFromFile(params, fileDataBase64);
        } else {
            params.content = document.getElementById('kb-content').value;
            await window.go.application.App.CreateSkill(params);
        }
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to create skill: ' + err);
    }
}

async function showEditSkillModal(skill) {
    let pdfSection = '';
    if (skill.contentType === 'pdf') {
        let pdfEmbed = '<p>No PDF file</p>';
        try {
            const base64 = await window.go.application.App.GetSkillPDF(skill.id);
            if (base64) {
                const dataUri = `data:application/pdf;base64,${base64}`;
                pdfEmbed = `<embed src="${dataUri}" type="application/pdf" width="100%" height="300px" style="border: 1px solid #ccc; border-radius: 4px;">`;
            }
        } catch (e) {
            console.error('Failed to load PDF:', e);
        }
        pdfSection = `
        <div class="form-group">
            <label>Current PDF</label>
            ${pdfEmbed}
        </div>
        <div class="form-group">
            <label for="kb-file">Replace PDF</label>
            <input type="file" id="kb-file" accept=".pdf">
        </div>`;
    }

    const body = `
        <div class="form-group">
            <label for="kb-title">Title *</label>
            <input type="text" id="kb-title" value="${escapeHtml(skill.title)}" placeholder="Title (required)" disabled>
        </div>
        ${pdfSection}
        <div class="form-group" id="kb-content-group" style="${skill.contentType === 'pdf' ? 'display:none' : ''}">
            <label for="kb-content">Skill</label>
            <textarea id="kb-content" placeholder="Skill" required>${escapeHtml(skill.content || '')}</textarea>
        </div>`;

    let skillId = skill.id;
    showModal('Edit Skill', body, () => submitUpdateSkill(skillId, skill.contentType), 'Save');
}

async function submitUpdateSkill(skillId, contentType) {
    const params = {
        title: document.getElementById('kb-title').value,
    };

    try {
        if (contentType === 'pdf') {
            const fileInput = document.getElementById('kb-file');
            const file = fileInput.files[0];

            if (file) {
                const arrayBuffer = await file.arrayBuffer();
                const bytes = new Uint8Array(arrayBuffer);
                let binary = '';
                for (let i = 0; i < bytes.byteLength; i++) {
                    binary += String.fromCharCode(bytes[i]);
                }
                const fileDataBase64 = btoa(binary);

                await window.go.application.App.UpdateSkillFromFile(skillId, params, fileDataBase64);
            } else {
                params.content = document.getElementById('kb-content').value;
                await window.go.application.App.UpdateSkill(skillId, params);
            }
        } else {
            params.content = document.getElementById('kb-content').value;
            await window.go.application.App.UpdateSkill(skillId, params);
        }
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to update skill: ' + err);
    }
}

function confirmDeleteSkill(skillId, skillTitle) {
    const body = `<p>Are you sure you want to delete the ${escapeHtml(skillTitle)}?</p>`;
    showModal('Delete Skill', body, () => deleteSkill(skillId), 'Delete');
}

async function deleteSkill(skillId) {
    try {
        console.log("deleting skill: ",skillId)
        await window.go.application.App.DeleteSkill(skillId);
        hideModal();
        renderView();
    } catch (err) {
        showToast('Failed to delete skill: ' + err);
    }
}

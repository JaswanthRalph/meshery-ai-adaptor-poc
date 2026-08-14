// Copyright 2026 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ═══════════════════════════════════════════════════════════════
// Meshery AI Adapter PoC — Frontend Application
// CNCF LFX Mentorship 2026 Term 3
// ═══════════════════════════════════════════════════════════════

const API = '';

// ─── State ───
let providers = [];
let connections = [];
let wizardStep = 1;
let selectedProvider = null;
let lastDesign = null;

// ─── Init ───
document.addEventListener('DOMContentLoaded', async () => {
  await loadProviders();
  await loadConnections();
  updateProviderSelect();

  document.getElementById('promptInput').addEventListener('input', updateGenerateBtn);
  document.getElementById('providerSelect').addEventListener('change', updateGenerateBtn);
});

// ─── API Helpers ───
async function apiFetch(path, options = {}) {
  const res = await fetch(API + path, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  return res.json();
}

// ─── Providers ───
async function loadProviders() {
  providers = await apiFetch('/api/ai/providers');
}

// ─── Connections ───
async function loadConnections() {
  connections = await apiFetch('/api/ai/connections');
  if (!Array.isArray(connections)) connections = [];
  renderConnections();
  updateProviderSelect();
}

function renderConnections() {
  const list = document.getElementById('connectionList');
  const empty = document.getElementById('emptyState');

  if (connections.length === 0) {
    list.innerHTML = '';
    list.appendChild(empty);
    empty.style.display = 'block';
    return;
  }

  empty.style.display = 'none';
  list.innerHTML = connections.map(conn => {
    const kindClass = `conn-kind-${conn.kind}`;
    const statusClass = conn.status || 'registered';
    const providerInfo = providers.find(p => p.kind === conn.kind);
    const providerName = providerInfo ? providerInfo.name : conn.kind;
    const model = conn.config?.model || conn.config?.deployment_id || '—';

    return `
      <div class="connection-card" data-status="${statusClass}" data-id="${conn.id}">
        <div class="conn-header">
          <span class="conn-name">
            <span class="status-dot ${statusClass}"></span>
            ${escHtml(conn.name)}
          </span>
          <span class="conn-kind ${kindClass}">${providerName}</span>
        </div>
        <div class="conn-meta">
          <span>Model: ${escHtml(model)}</span>
          <span>Status: ${statusClass}</span>
        </div>
        <div class="conn-actions">
          <button class="conn-action-btn" onclick="healthCheck('${conn.id}')">🩺 Health Check</button>
          <button class="conn-action-btn delete" onclick="deleteConnection('${conn.id}')">🗑 Delete</button>
        </div>
      </div>
    `;
  }).join('');
}

function updateProviderSelect() {
  const select = document.getElementById('providerSelect');
  select.innerHTML = '<option value="">Select AI Connection...</option>';
  connections.forEach(conn => {
    const opt = document.createElement('option');
    opt.value = conn.id;
    opt.textContent = `${conn.name} (${conn.kind})`;
    select.appendChild(opt);
  });
  updateGenerateBtn();
}

function updateGenerateBtn() {
  const prompt = document.getElementById('promptInput').value.trim();
  const connId = document.getElementById('providerSelect').value;
  document.getElementById('generateBtn').disabled = !prompt || !connId;
}

async function healthCheck(connId) {
  const card = document.querySelector(`.connection-card[data-id="${connId}"]`);
  const actionsDiv = card.querySelector('.conn-actions');
  const origHTML = actionsDiv.innerHTML;
  actionsDiv.innerHTML = '<div class="spinner" style="width:16px;height:16px;border-width:2px"></div> <span style="font-size:11px;color:var(--text-secondary)">Checking...</span>';

  try {
    const result = await apiFetch(`/api/ai/connections/${connId}/health`);
    await loadConnections();

    const statusEmoji = result.status === 'connected' ? '✅' : result.status === 'auth_failed' ? '🔑' : '❌';
    actionsDiv.innerHTML = `<span style="font-size:12px">${statusEmoji} ${result.status} (${result.latency_ms}ms)</span>`;
    setTimeout(() => { actionsDiv.innerHTML = origHTML; }, 4000);
  } catch (err) {
    actionsDiv.innerHTML = `<span style="font-size:12px;color:var(--accent-red)">❌ Failed</span>`;
    setTimeout(() => { actionsDiv.innerHTML = origHTML; }, 3000);
  }
}

async function deleteConnection(connId) {
  if (!confirm('Delete this connection?')) return;
  await apiFetch(`/api/ai/connections/${connId}`, { method: 'DELETE' });
  await loadConnections();
}

// ─── Wizard ───
function showWizard() {
  wizardStep = 1;
  selectedProvider = null;
  document.getElementById('wizardOverlay').style.display = 'flex';
  renderWizardStep();
  renderProviderGrid();
}

function closeWizard(event) {
  if (event && event.target !== event.currentTarget) return;
  document.getElementById('wizardOverlay').style.display = 'none';
}

function renderProviderGrid() {
  const grid = document.getElementById('providerGrid');
  const icons = { openai: '🧠', anthropic: '🔮', ollama: '🦙', 'azure-openai': '☁️', 'vertex-ai': '☁️' };

  grid.innerHTML = providers.map(p => `
    <div class="provider-card ${selectedProvider?.kind === p.kind ? 'selected' : ''}" 
         onclick="selectProvider('${p.kind}')">
      <div class="provider-card-icon">${icons[p.kind] || '🤖'}</div>
      <div class="provider-card-name">${p.name}</div>
      <div class="provider-card-desc">${p.description}</div>
    </div>
  `).join('');
}

function selectProvider(kind) {
  selectedProvider = providers.find(p => p.kind === kind);
  renderProviderGrid();
}

function renderWizardStep() {
  [1,2,3,4].forEach(i => {
    document.getElementById(`wizStep${i}`).style.display = i === wizardStep ? 'block' : 'none';
  });

  document.querySelectorAll('.progress-step').forEach(el => {
    const step = parseInt(el.dataset.step);
    el.classList.toggle('active', step === wizardStep);
    el.classList.toggle('done', step < wizardStep);
  });

  document.getElementById('wizBackBtn').style.display = wizardStep > 1 ? 'inline-flex' : 'none';

  const nextBtn = document.getElementById('wizNextBtn');
  if (wizardStep === 4) {
    nextBtn.textContent = '✓ Done';
  } else if (wizardStep === 3) {
    nextBtn.textContent = 'Create & Verify →';
  } else {
    nextBtn.textContent = 'Next →';
  }
}

function wizardNext() {
  if (wizardStep === 1) {
    if (!selectedProvider) { alert('Please select a provider'); return; }
    wizardStep = 2;
    renderConfigFields();
  } else if (wizardStep === 2) {
    if (!document.getElementById('connName').value.trim()) {
      alert('Please enter a connection name'); return;
    }
    wizardStep = 3;
    renderCredFields();
  } else if (wizardStep === 3) {
    wizardStep = 4;
    createConnectionAndVerify();
  } else if (wizardStep === 4) {
    closeWizard();
    loadConnections();
  }
  renderWizardStep();
}

function wizardBack() {
  if (wizardStep > 1) { wizardStep--; renderWizardStep(); }
}

function renderConfigFields() {
  const container = document.getElementById('configFields');
  const defaults = selectedProvider.default_config || {};

  container.innerHTML = selectedProvider.config_fields.map(field => `
    <div class="form-group">
      <label for="config_${field}">${formatLabel(field)}</label>
      <input type="text" id="config_${field}" class="form-input" 
             placeholder="${defaults[field] || ''}" value="${defaults[field] || ''}">
    </div>
  `).join('');
}

function renderCredFields() {
  const container = document.getElementById('credFields');

  if (!selectedProvider.requires_creds || selectedProvider.cred_fields.length === 0) {
    container.innerHTML = `
      <div class="security-notice" style="background:rgba(88,166,255,0.06);border-color:rgba(88,166,255,0.15)">
        <span class="security-icon">ℹ️</span>
        <div>
          <strong>No credentials required</strong>
          <p>This provider runs locally and doesn't need API keys. Your data stays on your machine.</p>
        </div>
      </div>`;
    return;
  }

  container.innerHTML = selectedProvider.cred_fields.map(field => `
    <div class="form-group">
      <label for="cred_${field}">${formatLabel(field)}</label>
      <input type="password" id="cred_${field}" class="form-input" 
             placeholder="Enter your ${formatLabel(field).toLowerCase()}">
    </div>
  `).join('');
}

async function createConnectionAndVerify() {
  const area = document.getElementById('healthCheckArea');
  area.innerHTML = '<div class="health-checking"><div class="spinner"></div><p>Creating connection and running health check...</p></div>';

  try {
    // Step 1: Create credential (if needed)
    let credentialId = '';
    if (selectedProvider.requires_creds && selectedProvider.cred_fields.length > 0) {
      const secret = {};
      selectedProvider.cred_fields.forEach(field => {
        secret[field] = document.getElementById(`cred_${field}`)?.value || '';
      });

      const credResult = await apiFetch('/api/ai/credentials', {
        method: 'POST',
        body: JSON.stringify({
          name: `${document.getElementById('connName').value} credential`,
          kind: selectedProvider.kind,
          secret: secret,
        }),
      });
      credentialId = credResult.id;
    }

    // Step 2: Create connection
    const config = {};
    selectedProvider.config_fields.forEach(field => {
      const val = document.getElementById(`config_${field}`)?.value;
      if (val) config[field] = val;
    });

    const connResult = await apiFetch('/api/ai/connections', {
      method: 'POST',
      body: JSON.stringify({
        name: document.getElementById('connName').value,
        kind: selectedProvider.kind,
        config: config,
        credential_id: credentialId,
      }),
    });

    // Step 3: Run health check
    const healthResult = await apiFetch(`/api/ai/connections/${connResult.id}/health`);

    const isSuccess = healthResult.status === 'connected';
    area.innerHTML = `
      <div class="health-result ${isSuccess ? 'success' : 'failure'}">
        <div class="health-result-icon">${isSuccess ? '✅' : '❌'}</div>
        <div class="health-result-status">${healthResult.status.replace('_', ' ').toUpperCase()}</div>
        <div class="health-result-msg">${escHtml(healthResult.message || '')}</div>
        <div class="health-result-latency">${healthResult.model_info ? `Model: ${healthResult.model_info} · ` : ''}Latency: ${healthResult.latency_ms}ms</div>
      </div>
    `;

    await loadConnections();
  } catch (err) {
    area.innerHTML = `
      <div class="health-result failure">
        <div class="health-result-icon">❌</div>
        <div class="health-result-status">ERROR</div>
        <div class="health-result-msg">${escHtml(err.message)}</div>
      </div>
    `;
  }
}

// ─── Generation ───
async function generateDesign() {
  const prompt = document.getElementById('promptInput').value.trim();
  const connId = document.getElementById('providerSelect').value;
  if (!prompt || !connId) return;

  const btn = document.getElementById('generateBtn');
  const resultDiv = document.getElementById('genResult');
  btn.disabled = true;
  btn.innerHTML = '<div class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle"></div> Generating...';

  resultDiv.style.display = 'none';

  try {
    const response = await fetch(API + '/api/ai/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Accept': 'text/event-stream' },
      body: JSON.stringify({ prompt, connection_id: connId }),
    });

    if (!response.ok) throw new Error(`HTTP error: ${response.status}`);

    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n\n');
      buffer = lines.pop(); // Keep the last incomplete chunk

      for (const chunk of lines) {
        const line = chunk.trim();
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.substring(6));
            if (data.status) {
              btn.innerHTML = `<div class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle"></div> ${data.status}`;
            } else if (data.token) {
              const yamlOut = document.getElementById('yamlOutput');
              if (resultDiv.style.display === 'none') {
                resultDiv.style.display = 'block';
                document.getElementById('componentsGrid').innerHTML = '<div style="color:var(--text-muted);font-size:13px">Streaming...</div>';
                document.getElementById('yamlOutput').parentElement.open = true; // Auto open details
                yamlOut.textContent = '';
              }
              yamlOut.textContent += data.token;
            } else if (data.result) {
              lastDesign = data.result;
              renderGenerationResult(data.result);
            } else if (data.error) {
              throw new Error(data.error);
            }
          } catch (e) {
            // Ignore parse errors from partial streams if any
          }
        }
      }
    }
  } catch (err) {
    resultDiv.style.display = 'block';
    resultDiv.innerHTML = `<div class="validation-item error">❌ Generation failed: ${escHtml(err.message)}</div>`;
  } finally {
    btn.disabled = false;
    btn.innerHTML = '<span class="btn-icon">⚡</span> Generate Design';
    updateGenerateBtn();
  }
}

function renderGenerationResult(result) {
  const resultDiv = document.getElementById('genResult');
  resultDiv.style.display = 'block';

  // Title & Meta
  document.getElementById('resultTitle').textContent = result.design?.name
    ? `Design: ${result.design.name}`
    : (result.success ? 'Generated Design' : 'Generation Failed');

  document.getElementById('resultMeta').innerHTML = `
    <span>Provider: ${result.provider_kind}</span>
    <span>Model: ${result.model_used || '—'}</span>
    <span>${result.latency_ms}ms</span>
    <span>${result.success ? '✅ Valid' : '⚠️ Issues'}</span>
  `;

  // Components
  const grid = document.getElementById('componentsGrid');
  if (result.design?.components?.length) {
    grid.innerHTML = result.design.components.map(comp => {
      const details = [];
      if (comp.config?.replicas) details.push(`${comp.config.replicas} replicas`);
      if (comp.config?.containers?.[0]?.image) details.push(comp.config.containers[0].image);
      if (comp.config?.type) details.push(comp.config.type);
      if (comp.config?.ports) details.push(`port ${comp.config.ports.map(p => p.port || p.containerPort).join(', ')}`);
      if (comp.namespace) details.push(`ns: ${comp.namespace}`);

      return `
        <div class="component-card">
          <div class="comp-kind comp-kind-${comp.kind}">${comp.kind}</div>
          <div class="comp-name">${escHtml(comp.name)}</div>
          <div class="comp-details">${details.map(escHtml).join(' · ') || comp.apiVersion}</div>
        </div>
      `;
    }).join('');
  } else {
    grid.innerHTML = '<div style="color:var(--text-muted);font-size:13px">No components generated</div>';
  }

  // Relationships
  const relSection = document.getElementById('relationshipsSection');
  const relList = document.getElementById('relationshipsList');
  if (result.design?.relationships?.length) {
    relSection.style.display = 'block';
    relList.innerHTML = result.design.relationships.map(rel => `
      <div class="relationship-chip">
        ${escHtml(rel.source)} <span class="rel-arrow">→</span> ${escHtml(rel.target)}
        <span style="color:var(--text-muted);font-size:10px">(${rel.type})</span>
      </div>
    `).join('');
  } else {
    relSection.style.display = 'none';
  }

  // Validation
  const valSection = document.getElementById('validationSection');
  const valList = document.getElementById('validationList');
  if (result.validation_errors?.length) {
    valSection.style.display = 'block';
    valList.innerHTML = result.validation_errors.map(err => `
      <div class="validation-item ${err.severity}">
        ${err.severity === 'error' ? '❌' : '⚠️'}
        <span><strong>${escHtml(err.component)}</strong>: ${escHtml(err.message)}</span>
      </div>
    `).join('');
  } else {
    valSection.style.display = 'none';
  }

  // YAML
  const yamlOut = document.getElementById('yamlOutput');
  if (result.design) {
    yamlOut.textContent = JSON.stringify(result.design, null, 2);
  } else if (result.raw_output) {
    yamlOut.textContent = result.raw_output;
  } else if (!yamlOut.textContent) {
    yamlOut.textContent = JSON.stringify(result, null, 2);
  }

  // Kanvas Button
  const kanvasBtn = document.getElementById('kanvasBtn');
  if (result.success && result.design) {
    kanvasBtn.disabled = false;
    kanvasBtn.title = "Export to Meshery Kanvas";
  } else {
    kanvasBtn.disabled = true;
    kanvasBtn.title = "Must generate a valid design first";
  }
}

async function openInKanvas() {
  if (!lastDesign?.design) return;
  const btn = document.getElementById('kanvasBtn');
  btn.innerHTML = '<div class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle"></div> Opening...';
  try {
    const payload = {
      pattern_data: JSON.stringify(lastDesign.design),
      name: lastDesign.design.name || "AI-Generated Design"
    };
    const res = await fetch('http://localhost:9081/api/pattern', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) throw new Error("Failed to export to Meshery Server");
    const data = await res.json();
    const patternId = data.id || data.pattern_id || data[0]?.id || "";
    window.open(`http://localhost:9081/extension/meshery-kanvas?design_id=${patternId}`, '_blank');
  } catch (e) {
    alert("Failed to export to Kanvas. Ensure Meshery server is running at localhost:9081.\n\n" + e.message);
  } finally {
    btn.innerHTML = '🎨 Open in Kanvas';
  }
}

function copyDesign() {
  if (!lastDesign?.design) return;
  navigator.clipboard.writeText(JSON.stringify(lastDesign.design, null, 2));
  const btn = event.target;
  btn.textContent = '✓ Copied!';
  setTimeout(() => { btn.textContent = '📋 Copy Design'; }, 2000);
}

function downloadDesign() {
  if (!lastDesign?.design) return;
  const blob = new Blob([JSON.stringify(lastDesign.design, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${lastDesign.design.name || 'meshery-design'}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

// ─── Credentials Modal ───
async function showCredentials() {
  const modal = document.getElementById('credModal');
  modal.style.display = 'flex';

  const list = document.getElementById('credList');
  list.innerHTML = '<div class="health-checking"><div class="spinner"></div></div>';

  const creds = await apiFetch('/api/ai/credentials');
  if (!Array.isArray(creds) || creds.length === 0) {
    list.innerHTML = '<div class="empty-state" style="padding:24px"><div class="empty-icon">🔑</div><h3>No Credentials</h3><p>Create a connection to store credentials.</p></div>';
    return;
  }

  list.innerHTML = creds.map(c => `
    <div class="cred-item">
      <div class="cred-item-info">
        <span class="cred-item-name">${escHtml(c.name)}</span>
        <span class="cred-item-kind">${c.kind} · Created ${new Date(c.created_at).toLocaleDateString()}</span>
      </div>
      <div class="cred-item-secrets">
        ${Object.keys(c.has_secret || {}).map(k => 
          `<span class="cred-secret-badge">🔒 ${k}</span>`
        ).join('')}
      </div>
    </div>
  `).join('');
}

function closeCredModal(event) {
  if (event && event.target !== event.currentTarget) return;
  document.getElementById('credModal').style.display = 'none';
}

// ─── Helpers ───
function formatLabel(field) {
  return field.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
}

function escHtml(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

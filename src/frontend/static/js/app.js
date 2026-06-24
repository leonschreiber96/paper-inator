// paper-inator frontend — pure vanilla JS, no build step.
// Sections: utils → api → navigation → feeds → mappings → publications → summaries → settings → init

// ── Utils ─────────────────────────────────────────────────────────────────────

const el = (id) => document.getElementById(id);

function fmtDate(value) {
  if (!value) return '';
  const d = new Date(value);
  return isNaN(d) ? '' : d.toLocaleDateString();
}

function escapeHTML(s) {
  return String(s ?? '').replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
function escapeAttr(s) { return escapeHTML(s); }

function debounce(fn, ms) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

function showError(id, msg) {
  const p = el(id);
  p.textContent = msg;
  p.hidden = false;
}
function clearError(id) { el(id).hidden = true; }

// ── API wrapper ────────────────────────────────────────────────────────────────

const api = {
  async get(path) {
    const res = await fetch(path);
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
    return res.json();
  },
  async send(method, path, body) {
    const res = await fetch(path, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
    if (res.status === 204 || res.status === 202) return null;
    const text = await res.text();
    return text ? JSON.parse(text) : null;
  },
};

// ── Navigation ─────────────────────────────────────────────────────────────────

const SECTIONS = document.querySelectorAll('section[data-hash]');
const TAB_BTNS = document.querySelectorAll('.tab-btn');

function activateTab(hash) {
  const h = hash || '#feeds';
  SECTIONS.forEach((s) => s.classList.toggle('hidden', s.dataset.hash !== h));
  TAB_BTNS.forEach((b) => b.classList.toggle('active', b.dataset.hash === h));
  // Load section data on first visit
  if (h === '#summaries') loadSummaries();
  if (h === '#settings') loadSettings();
}

TAB_BTNS.forEach((btn) => btn.addEventListener('click', () => {
  location.hash = btn.dataset.hash;
}));
window.addEventListener('hashchange', () => activateTab(location.hash));

// ── Feeds ──────────────────────────────────────────────────────────────────────

// Cached feed list (shared with summaries form checkboxes)
let feedCache = [];

async function loadFeeds() {
  feedCache = await api.get('/api/feeds');
  renderFeeds();
  syncFeedFilter();
}

function renderFeeds() {
  const list = el('feed-list');
  list.innerHTML = '';
  if (feedCache.length === 0) {
    list.innerHTML = '<li class="empty">No feeds yet. Add one above.</li>';
    return;
  }
  for (const f of feedCache) {
    list.append(buildFeedItem(f));
  }
}

function buildFeedItem(f) {
  const li = document.createElement('li');

  // Main row
  const row = document.createElement('div');
  row.className = 'feed-row';

  const info = document.createElement('div');
  info.className = 'feed-info';
  info.innerHTML =
    `<div><strong>${escapeHTML(f.name)}</strong></div>` +
    `<div class="feed-meta">${escapeHTML(f.url)}` +
    (f.last_fetched_at ? ` · last fetched ${fmtDate(f.last_fetched_at)}` : ' · not fetched yet') +
    `</div>`;

  const actions = document.createElement('div');
  actions.className = 'feed-actions';

  // Enabled toggle
  const toggleLabel = document.createElement('label');
  toggleLabel.className = 'feed-toggle';
  toggleLabel.title = f.enabled ? 'Enabled — click to disable' : 'Disabled — click to enable';
  const chk = document.createElement('input');
  chk.type = 'checkbox';
  chk.checked = f.enabled;
  chk.addEventListener('change', async () => {
    try {
      await api.send('PUT', `/api/feeds/${f.id}`, { ...f, enabled: chk.checked });
      f.enabled = chk.checked;
      toggleLabel.title = f.enabled ? 'Enabled — click to disable' : 'Disabled — click to enable';
    } catch (err) {
      chk.checked = f.enabled; // revert on error
      alert(err.message);
    }
  });
  const track = document.createElement('span');
  track.className = 'toggle-track';
  const tlabel = document.createElement('span');
  tlabel.className = 'toggle-label';
  tlabel.textContent = 'enabled';
  toggleLabel.append(chk, track, tlabel);

  // Delete button
  const del = document.createElement('button');
  del.className = 'link';
  del.textContent = 'Delete';
  del.onclick = async () => {
    if (!confirm(`Delete feed "${f.name}"? Its publications will be removed.`)) return;
    try {
      await api.send('DELETE', `/api/feeds/${f.id}`);
      await refresh();
    } catch (err) { alert(err.message); }
  };

  actions.append(toggleLabel, del);
  row.append(info, actions);
  li.append(row);

  // Expandable details (edit form + mappings)
  const details = document.createElement('details');
  details.className = 'feed-details';
  const summ = document.createElement('summary');
  summ.textContent = 'Edit / Mappings';
  const { el: mappingEl, load: loadMapping } = buildMappingsEditor(f);
  details.append(summ, buildFeedEditForm(f), mappingEl);
  details.addEventListener('toggle', () => { if (details.open) loadMapping(); }, { once: true });
  li.append(details);

  return li;
}

function buildFeedEditForm(f) {
  const wrap = document.createElement('div');
  wrap.innerHTML = `
    <form class="feed-edit-form" data-feed-id="${f.id}">
      <label>Name
        <input type="text" name="name" value="${escapeAttr(f.name)}" required>
      </label>
      <label>URL
        <input type="url" name="url" value="${escapeAttr(f.url)}" required>
      </label>
      <label>Fetch interval (sec, 0 = global default)
        <input type="number" name="interval" value="${f.fetch_interval_sec}" min="0">
      </label>
      <div class="form-actions">
        <button type="submit">Save</button>
        <button type="reset" class="secondary">Reset</button>
      </div>
    </form>`;
  wrap.querySelector('form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    try {
      await api.send('PUT', `/api/feeds/${f.id}`, {
        ...f,
        name: form.name.value.trim(),
        url: form.url.value.trim(),
        fetch_interval_sec: parseInt(form.interval.value, 10) || 0,
      });
      await loadFeeds();
    } catch (err) { alert(err.message); }
  });
  return wrap;
}

// ── Field Mappings (drag-and-drop) ────────────────────────────────────────────

const TARGET_FIELDS = ['title', 'authors', 'abstract', 'link', 'published_at'];
const TARGET_LABELS = { title: 'Title', authors: 'Authors', abstract: 'Abstract', link: 'Link', published_at: 'Published' };

// buildMappingsEditor returns { el, load } where el is the DOM node and load()
// fetches fields + current mappings and renders the DnD UI. load() is called by
// buildFeedItem on first open of the <details> panel.
function buildMappingsEditor(f) {
  let discoveredFields = []; // [{field_name, sample_value}]
  let assignments = Object.fromEntries(TARGET_FIELDS.map((t) => [t, null]));

  const wrap = document.createElement('div');
  wrap.className = 'mappings-wrap';
  wrap.innerHTML = `
    <div class="field-pool-section">
      <div class="field-pool-label">Feed fields</div>
      <div class="field-pool"></div>
    </div>
    <div class="assignment-grid-section">
      <div class="field-pool-label">Target assignments</div>
      <div class="assignment-grid"></div>
    </div>
    <div class="mappings-actions"></div>`;

  const poolEl = wrap.querySelector('.field-pool');

  // Pool accepts drops from target zones (returns chip to unassigned pool)
  poolEl.addEventListener('dragover', (e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; poolEl.classList.add('drag-over'); });
  poolEl.addEventListener('dragleave', (e) => { if (!poolEl.contains(e.relatedTarget)) poolEl.classList.remove('drag-over'); });
  poolEl.addEventListener('drop', (e) => {
    e.preventDefault();
    poolEl.classList.remove('drag-over');
    const fromTarget = e.dataTransfer.getData('from-target');
    if (fromTarget) { assignments[fromTarget] = null; rerender(); }
  });

  // Actions
  const actionsEl = wrap.querySelector('.mappings-actions');
  const refreshBtn = document.createElement('button');
  refreshBtn.type = 'button';
  refreshBtn.className = 'secondary';
  refreshBtn.textContent = '↻ Refresh fields';
  refreshBtn.onclick = async () => {
    refreshBtn.disabled = true;
    refreshBtn.textContent = 'Refreshing…';
    try {
      await api.send('POST', `/api/feeds/${f.id}/ingest`);
      // Give the async worker time to finish before re-fetching fields
      await new Promise((r) => setTimeout(r, 2500));
      await load();
    } catch (err) { alert(err.message); }
    finally { refreshBtn.disabled = false; refreshBtn.textContent = '↻ Refresh fields'; }
  };

  const saveBtn = document.createElement('button');
  saveBtn.type = 'button';
  saveBtn.textContent = 'Save mappings';
  saveBtn.onclick = async () => {
    const mappings = TARGET_FIELDS
      .filter((t) => assignments[t])
      .map((t) => ({ source_field: assignments[t], target_field: t }));
    try {
      await api.send('PUT', `/api/feeds/${f.id}/mappings`, mappings);
      saveBtn.textContent = '✓ Saved';
      setTimeout(() => { saveBtn.textContent = 'Save mappings'; }, 1500);
    } catch (err) { alert(err.message); }
  };
  actionsEl.append(refreshBtn, saveBtn);

  function makeChip(fieldName, sampleValue, fromTarget) {
    const chip = document.createElement('div');
    chip.className = 'field-chip';
    chip.draggable = true;
    chip.title = sampleValue ? `${fieldName}\nSample: ${sampleValue}` : fieldName;
    const nameSpan = document.createElement('span');
    nameSpan.textContent = fieldName;
    chip.append(nameSpan);
    if (fromTarget) {
      const rmBtn = document.createElement('button');
      rmBtn.type = 'button';
      rmBtn.className = 'chip-remove';
      rmBtn.textContent = '×';
      rmBtn.onclick = (e) => { e.stopPropagation(); assignments[fromTarget] = null; rerender(); };
      chip.append(rmBtn);
    }
    chip.addEventListener('dragstart', (e) => {
      e.dataTransfer.setData('text/plain', fieldName);
      e.dataTransfer.setData('from-target', fromTarget || '');
      e.dataTransfer.effectAllowed = 'move';
      chip.classList.add('dragging');
    });
    chip.addEventListener('dragend', () => chip.classList.remove('dragging'));
    return chip;
  }

  function makeDropZone(target) {
    const zone = document.createElement('div');
    zone.className = 'drop-zone';
    zone.addEventListener('dragover', (e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; zone.classList.add('drag-over'); });
    zone.addEventListener('dragleave', (e) => { if (!zone.contains(e.relatedTarget)) zone.classList.remove('drag-over'); });
    zone.addEventListener('drop', (e) => {
      e.preventDefault();
      zone.classList.remove('drag-over');
      const fieldName = e.dataTransfer.getData('text/plain');
      const fromTarget = e.dataTransfer.getData('from-target');
      if (!fieldName) return;
      if (fromTarget && fromTarget !== target) assignments[fromTarget] = null;
      assignments[target] = fieldName;
      rerender();
    });
    return zone;
  }

  function rerender() {
    const assigned = new Set(Object.values(assignments).filter(Boolean));

    // Pool
    poolEl.innerHTML = '';
    const unassigned = discoveredFields.filter((df) => !assigned.has(df.field_name));
    if (discoveredFields.length === 0) {
      poolEl.innerHTML = '<span class="empty" style="font-size:0.8rem">No fields discovered yet — click ↻ Refresh fields.</span>';
    } else {
      for (const df of unassigned) poolEl.append(makeChip(df.field_name, df.sample_value, null));
    }

    // Assignment grid
    const gridEl = wrap.querySelector('.assignment-grid');
    gridEl.innerHTML = '';
    for (const target of TARGET_FIELDS) {
      const row = document.createElement('div');
      row.className = 'assignment-row';
      const lbl = document.createElement('div');
      lbl.className = 'assignment-label';
      lbl.textContent = TARGET_LABELS[target];
      const zone = makeDropZone(target);
      const src = assignments[target];
      if (src) {
        const df = discoveredFields.find((d) => d.field_name === src);
        zone.append(makeChip(src, df?.sample_value ?? '', target));
        zone.classList.add('has-chip');
      }
      row.append(lbl, zone);
      gridEl.append(row);
    }
  }

  async function load() {
    try {
      const [fields, mappings] = await Promise.all([
        api.get(`/api/feeds/${f.id}/fields`),
        api.get(`/api/feeds/${f.id}/mappings`),
      ]);
      discoveredFields = fields;
      for (const t of TARGET_FIELDS) assignments[t] = null;
      for (const m of mappings) {
        if (TARGET_FIELDS.includes(m.target_field)) assignments[m.target_field] = m.source_field;
      }
      rerender();
    } catch (err) { alert(err.message); }
  }

  rerender(); // empty initial state before load() is called
  return { el: wrap, load };
}

// ── Publications ───────────────────────────────────────────────────────────────

const PAGE_SIZE = 25;
let pubOffset = 0;
let pubTotal = 0; // track whether there may be more

function syncFeedFilter() {
  const sel = el('filter-feed');
  const cur = sel.value;
  sel.length = 1; // keep "All feeds"
  for (const f of feedCache) {
    const opt = new Option(f.name, f.id);
    if (String(f.id) === cur) opt.selected = true;
    sel.append(opt);
  }
}

async function loadPublications(reset = true) {
  if (reset) {
    pubOffset = 0;
    el('publication-list').innerHTML = '';
  }
  const params = new URLSearchParams({
    limit: PAGE_SIZE,
    offset: pubOffset,
    sort: el('filter-sort').value,
    order: 'desc',
  });
  const feedID = el('filter-feed').value;
  const q = el('filter-q').value.trim();
  if (feedID) params.set('feed_id', feedID);
  if (q) params.set('q', q);

  const pubs = await api.get('/api/publications?' + params.toString());
  pubTotal = pubs.length;
  renderPublications(pubs, reset);
  el('load-more-row').hidden = pubs.length < PAGE_SIZE;
}

function renderPublications(pubs, reset) {
  const list = el('publication-list');
  if (reset && pubs.length === 0) {
    list.innerHTML = '<li class="empty">No publications yet.</li>';
    return;
  }
  for (const p of pubs) {
    const li = document.createElement('li');
    const titleLink = p.link
      ? `<a href="${escapeAttr(p.link)}" target="_blank" rel="noopener">${escapeHTML(p.title)}</a>`
      : escapeHTML(p.title);
    const hasAbstract = p.abstract && p.abstract.trim();
    li.innerHTML =
      `<div class="pub-title">${titleLink}</div>` +
      `<div class="pub-meta">${escapeHTML(p.authors || 'Unknown authors')}` +
      (p.published_at ? ` · ${fmtDate(p.published_at)}` : '') +
      (hasAbstract ? ` <button class="pub-abstract-toggle" aria-expanded="false">▸ Abstract</button>` : '') +
      `</div>` +
      (hasAbstract ? `<div class="pub-abstract">${escapeHTML(p.abstract)}</div>` : '');
    if (hasAbstract) {
      const btn = li.querySelector('.pub-abstract-toggle');
      const div = li.querySelector('.pub-abstract');
      btn.addEventListener('click', () => {
        const open = div.classList.toggle('open');
        btn.textContent = open ? '▾ Abstract' : '▸ Abstract';
        btn.setAttribute('aria-expanded', open);
      });
    }
    list.append(li);
  }
}

el('load-more-btn').addEventListener('click', async () => {
  pubOffset += PAGE_SIZE;
  await loadPublications(false);
});

['filter-feed', 'filter-sort'].forEach((id) => el(id).addEventListener('change', () => loadPublications(true)));
el('filter-q').addEventListener('input', debounce(() => loadPublications(true), 250));

// ── Summaries ──────────────────────────────────────────────────────────────────

let summaryEditingId = null;

async function loadSummaries() {
  try {
    const summaries = await api.get('/api/summaries');
    renderSummaries(summaries);
  } catch (err) { alert(err.message); }
}

function renderSummaries(summaries) {
  const list = el('summary-list');
  list.innerHTML = '';
  if (summaries.length === 0) {
    list.innerHTML = '<li class="empty">No summaries yet. Create one above.</li>';
    return;
  }
  const feedMap = Object.fromEntries(feedCache.map((f) => [f.id, f.name]));
  for (const s of summaries) {
    const feedNames = (s.feed_ids || []).map((id) => feedMap[id] || `#${id}`).join(', ') || 'All feeds';
    const li = document.createElement('li');
    li.innerHTML = `
      <div class="summary-row">
        <div class="summary-info">
          <div><strong>${escapeHTML(s.name)}</strong>
            <span class="summary-badge ${s.enabled ? 'on' : ''}">${s.enabled ? 'enabled' : 'disabled'}</span>
          </div>
          <div class="summary-meta">→ ${escapeHTML(s.recipient)} · ${s.max_items} items · feeds: ${escapeHTML(feedNames)}</div>
        </div>
        <div class="summary-actions">
          <button class="secondary edit-summary">Edit</button>
          <button class="link del-summary">Delete</button>
        </div>
      </div>`;
    li.querySelector('.edit-summary').onclick = () => openSummaryForm(s);
    li.querySelector('.del-summary').onclick = async () => {
      if (!confirm(`Delete summary "${s.name}"?`)) return;
      try { await api.send('DELETE', `/api/summaries/${s.id}`); await loadSummaries(); }
      catch (err) { alert(err.message); }
    };
    list.append(li);
  }
}

function openSummaryForm(s) {
  summaryEditingId = s.id;
  el('summary-id').value = s.id;
  el('summary-name').value = s.name;
  el('summary-recipient').value = s.recipient;
  el('summary-max-items').value = s.max_items;
  el('summary-enabled').checked = s.enabled;
  el('summary-submit').textContent = 'Update';
  buildSummaryFeedChecks(s.feed_ids || []);
  el('summary-form-details').open = true;
}

function buildSummaryFeedChecks(selectedIds = []) {
  const container = el('summary-feed-checks');
  container.innerHTML = '';
  for (const f of feedCache) {
    const label = document.createElement('label');
    label.className = 'feed-check-label';
    const chk = document.createElement('input');
    chk.type = 'checkbox';
    chk.value = f.id;
    chk.checked = selectedIds.includes(f.id);
    label.append(chk, document.createTextNode(f.name));
    container.append(label);
  }
}

// Populate feed checkboxes when the summaries form is opened
el('summary-form-details').addEventListener('toggle', () => {
  if (el('summary-form-details').open) buildSummaryFeedChecks([]);
});

el('summary-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  clearError('summary-error');
  const selectedFeeds = [...el('summary-feed-checks').querySelectorAll('input:checked')]
    .map((c) => parseInt(c.value, 10));
  const payload = {
    name: el('summary-name').value.trim(),
    recipient: el('summary-recipient').value.trim(),
    max_items: parseInt(el('summary-max-items').value, 10) || 10,
    feed_ids: selectedFeeds,
    enabled: el('summary-enabled').checked,
    schedule: '',
  };
  try {
    if (summaryEditingId) {
      await api.send('PUT', `/api/summaries/${summaryEditingId}`, { ...payload, id: summaryEditingId });
    } else {
      await api.send('POST', '/api/summaries', payload);
    }
    resetSummaryForm();
    await loadSummaries();
  } catch (err) { showError('summary-error', err.message); }
});

el('summary-cancel').addEventListener('click', resetSummaryForm);

function resetSummaryForm() {
  summaryEditingId = null;
  el('summary-form').reset();
  el('summary-id').value = '';
  el('summary-submit').textContent = 'Save';
  el('summary-form-details').open = false;
  clearError('summary-error');
}

// ── Settings ───────────────────────────────────────────────────────────────────

async function loadSettings() {
  try {
    const s = await api.get('/api/settings/fetch_interval_minutes').catch(() => null);
    if (s) el('setting-fetch-interval').value = s.value;
  } catch (_) { /* setting may not exist yet */ }
}

el('settings-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  clearError('settings-error');
  el('settings-success').hidden = true;
  try {
    await api.send('PUT', '/api/settings/fetch_interval_minutes', {
      value: String(el('setting-fetch-interval').value || '0'),
    });
    el('settings-success').hidden = false;
  } catch (err) { showError('settings-error', err.message); }
});

// ── Global actions & init ──────────────────────────────────────────────────────

async function refresh() {
  await loadFeeds();
  await loadPublications(true);
}

el('add-feed-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  clearError('feed-error');
  try {
    await api.send('POST', '/api/feeds', {
      name: el('feed-name').value.trim(),
      url: el('feed-url').value.trim(),
      enabled: true,
      fetch_interval_sec: 0,
    });
    el('feed-name').value = '';
    el('feed-url').value = '';
    await refresh();
  } catch (err) { showError('feed-error', err.message); }
});

// Bootstrap
activateTab(location.hash || '#feeds');
refresh().catch(console.error);

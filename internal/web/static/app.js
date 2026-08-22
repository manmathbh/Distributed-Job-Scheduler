/* Distributed Job Scheduler dashboard — vanilla JS (no build step). */

/* ---------------- icons (lucide-style inline SVG) ---------------- */
const ICONS = {
  workflow: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="8" height="8" x="3" y="3" rx="2"/><path d="M7 11v4a2 2 0 0 0 2 2h4"/><rect width="8" height="8" x="13" y="13" rx="2"/></svg>',
  x: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',
  menu: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="6" y2="6"/><line x1="4" x2="20" y1="12" y2="12"/><line x1="4" x2="20" y1="18" y2="18"/></svg>',
  'layout-dashboard': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="7" height="9" x="3" y="3" rx="1"/><rect width="7" height="5" x="14" y="3" rx="1"/><rect width="7" height="9" x="14" y="12" rx="1"/><rect width="7" height="5" x="3" y="16" rx="1"/></svg>',
  layers: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z"/><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65"/><path d="m22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65"/></svg>',
  'list-ordered': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="10" x2="21" y1="6" y2="6"/><line x1="10" x2="21" y1="12" y2="12"/><line x1="10" x2="21" y1="18" y2="18"/><path d="M4 6h1v4"/><path d="M4 10h2"/><path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"/></svg>',
  server: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="2" rx="2"/><rect width="20" height="8" x="2" y="14" rx="2"/><line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/></svg>',
  archive: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/></svg>',
  'bar-chart-3': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/></svg>',
  folder: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>',
  plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>',
  pencil: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
  play: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="6 3 20 12 6 21 6 3"/></svg>',
  pause: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="14" y="4" width="4" height="16" rx="1"/><rect x="6" y="4" width="4" height="16" rx="1"/></svg>',
  'refresh-cw': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>',
  ban: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>',
  'check-circle': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><path d="m9 11 3 3L22 4"/></svg>',
  'alert-triangle': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>',
  activity: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>',
  inbox: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>',
  zap: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',
  'chevron-left': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',
  'chevron-right': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>'
};

function icon(name) { return ICONS[name] || ''; }

/* ---------------- api client ---------------- */
const api = {
  key: localStorage.getItem('scheduler_api_key') || '',
  base: '/api/v1',
  async request(path, opts = {}) {
    if (!this.key) throw new Error('no api key');
    const res = await fetch(this.base + path, {
      ...opts,
      headers: { 'Authorization': 'Bearer ' + this.key, 'Content-Type': 'application/json', ...(opts.headers || {}) }
    });
    if (res.status === 401) throw new Error('unauthorized');
    if (!res.ok) {
      let msg = res.statusText;
      try { const b = await res.json(); msg = b.error || msg; } catch (e) {}
      throw new Error(msg || ('HTTP ' + res.status));
    }
    return res.json();
  }
};

/* ---------------- state ---------------- */
const state = {
  route: 'overview',
  projects: [],
  currentProject: null,
  jobs: { queue_id: '', status: '', type: '', cursor: '', history: [] },
  dlq: { cursor: '', history: [] }
};

/* ---------------- helpers ---------------- */
const $ = (sel, root) => (root || document).querySelector(sel);
const $$ = (sel, root) => Array.from((root || document).querySelectorAll(sel));

function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
function num(n) { return (Number(n) || 0); }
function shortId(id) { return id ? id.slice(0, 8) : ''; }
function fmtTime(t) { return t ? new Date(t).toLocaleString() : '—'; }
function fmtDate(t) { return t ? new Date(t).toLocaleDateString() : '—'; }
function fmtRel(t) {
  if (!t) return '—';
  const s = (Date.now() - new Date(t).getTime()) / 1000;
  if (s < 0) return 'just now';
  if (s < 60) return Math.floor(s) + 's ago';
  if (s < 3600) return Math.floor(s / 60) + 'm ago';
  if (s < 86400) return Math.floor(s / 3600) + 'h ago';
  return Math.floor(s / 86400) + 'd ago';
}
function fmtDur(ms) { return ms == null || ms === '' ? '—' : (ms >= 1000 ? (ms / 1000).toFixed(1) + 's' : ms + 'ms'); }
function fmtUptime(sec) {
  if (sec == null) return '—';
  const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60);
  if (d > 0) return d + 'd ' + h + 'h';
  if (h > 0) return h + 'h ' + m + 'm';
  return m + 'm';
}

function badge(status) {
  const s = String(status || 'muted').toLowerCase();
  return '<span class="badge ' + esc(s) + '">' + esc(status || '—') + '</span>';
}

function applyIcons(root) {
  $$('[data-icon]', root).forEach((node) => {
    const name = node.getAttribute('data-icon');
    if (ICONS[name]) node.innerHTML = ICONS[name];
  });
}

function toast(msg, type) {
  const root = $('#toast-root');
  if (!root) return;
  const el = document.createElement('div');
  el.className = 'toast ' + (type === 'error' ? 'err' : type === 'ok' ? 'ok' : '');
  el.innerHTML = (type === 'error' ? icon('alert-triangle') : type === 'ok' ? icon('check-circle') : icon('activity')) + '<span>' + esc(msg) + '</span>';
  root.appendChild(el);
  setTimeout(() => el.remove(), 3500);
}

/* ---------------- router / render ---------------- */
const TITLES = { 'overview': 'Overview', 'queues': 'Queues', 'jobs': 'Jobs', 'workers': 'Workers', 'dead-letter': 'Dead Letter', 'metrics': 'Metrics' };
const PROJECT_ROUTES = ['queues', 'jobs', 'dead-letter'];

function currentRoute() {
  const h = location.hash.replace(/^#\/?/, '');
  return TITLES[h] ? h : 'overview';
}

function setConn(status) {
  const c = $('#connStatus');
  if (!c) return;
  c.classList.remove('ok', 'err');
  const t = $('.conn-text', c);
  if (status === 'ok') { c.classList.add('ok'); if (t) t.textContent = 'Connected'; }
  else if (status === 'err') { c.classList.add('err'); if (t) t.textContent = 'Unauthorized'; }
  else if (t) t.textContent = 'Not connected';
}

function updateNav(route) {
  $$('#nav .nav-item').forEach((a) => a.classList.toggle('active', a.getAttribute('data-route') === route));
  const t = $('#topbarTitle');
  if (t) t.textContent = TITLES[route] || '';
  const pp = $('#projectPickerWrap');
  if (pp) pp.hidden = !PROJECT_ROUTES.includes(route);
}

function skeleton() {
  return '<div class="skel-cards">' + '<div class="skeleton skel-card"></div>'.repeat(6) + '</div><div class="skeleton skel-row mt"></div>';
}

async function render() {
  const c = $('#content');
  if (!c) return;
  state.route = currentRoute();
  updateNav(state.route);

  if (!api.key) {
    setConn('');
    c.innerHTML = '<div class="empty">' + icon('zap') + '<div class="t">Connect to get started</div><div class="d">Enter your API key in the top bar to load scheduler data.</div></div>';
    return;
  }

  if (!c.innerHTML.trim()) c.innerHTML = skeleton();
  try {
    const html = await routes[state.route]();
    c.innerHTML = html;
  } catch (e) {
    if (e.message === 'unauthorized') setConn('err');
    c.innerHTML = '<div class="error-box">' + icon('alert-triangle') + '<div>' + esc(e.message) + '</div></div>';
    return;
  }
  setConn('ok');
}

/* ---------------- overview ---------------- */
async function overview() {
  const ov = await api.request('/overview');
  const jobs = ov.jobs || {};
  const m = ov.metrics || {};
  const total = Object.values(jobs).reduce((a, b) => a + num(b), 0);
  const running = num(jobs.running) + num(jobs.claimed);

  const stats = [
    ['Total Jobs', total, ''], ['Queued', num(jobs.queued), 'blue'], ['Running', running, 'amber'],
    ['Scheduled', num(jobs.scheduled), ''], ['Completed', num(jobs.completed), 'green'], ['Failed', num(jobs.failed), 'red'],
    ['Dead Lettered', num(ov.dead_lettered), 'red'], ['Projects', num(ov.projects), ''], ['Active Workers', num(ov.active_workers), 'green']
  ];

  const recent = await recentJobs();
  const health = await queueHealth();

  return '<div class="page-head"><div><div class="page-title">Overview</div><div class="page-sub">System-wide scheduler health and throughput.</div></div></div>' +
    '<div class="grid-cards">' + stats.map((s) =>
      '<div class="card stat"><div class="stat-top"><span class="stat-label">' + esc(s[0]) + '</span></div><div class="stat-value ' + s[2] + '">' + num(s[1]).toLocaleString() + '</div></div>'
    ).join('') + '</div>' +
    '<div class="two-col mt">' +
      '<div class="card"><div class="card-head"><div class="card-title">Job status distribution</div></div>' + donutChart(jobs, total) + '</div>' +
      '<div class="card"><div class="card-head"><div class="card-title">Throughput <span class="card-desc">· since start</span></div></div>' + throughputBars(m) + '</div>' +
    '</div>' +
    '<div class="two-col mt">' +
      '<div class="card"><div class="card-head"><div class="card-title">Recent jobs</div><div class="card-desc">' + (state.currentProject ? esc(state.currentProject.name) : '') + '</div></div>' + recent + '</div>' +
      '<div class="card"><div class="card-head"><div class="card-title">Queue health</div></div>' + health + '</div>' +
    '</div>';
}

async function recentJobs() {
  if (!state.projects.length) {
    try { await ensureProjects(); } catch (e) { return emptyState('inbox', 'No projects', 'Create a project to see recent jobs.'); }
  }
  const p = state.currentProject || state.projects[0];
  if (!p) return emptyState('inbox', 'No projects', 'Create a project to see recent jobs.');
  const data = await api.request('/projects/' + p.id + '/jobs?limit=8');
  const items = data.items || [];
  if (!items.length) return emptyState('inbox', 'No jobs yet', 'Jobs submitted to this project will appear here.');
  const rows = items.map((j) => '<tr class="clickable" onclick="showJob(\'' + esc(j.id) + '\')">' +
    '<td class="mono">' + esc(shortId(j.id)) + '</td><td>' + esc(j.type) + '</td><td>' + badge(j.status) + '</td>' +
    '<td>' + num(j.attempts) + '/' + num(j.max_attempts) + '</td><td class="cell-sub">' + fmtRel(j.created_at) + '</td></tr>').join('');
  return '<div class="table-scroll"><table><thead><tr><th>Job</th><th>Type</th><th>Status</th><th>Attempts</th><th>Created</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
}

async function queueHealth() {
  if (!state.projects.length) {
    try { await ensureProjects(); } catch (e) { return emptyState('layers', 'No projects', ''); }
  }
  const p = state.currentProject || state.projects[0];
  if (!p) return emptyState('layers', 'No projects', 'Create a project and queues to monitor health.');
  let queues = [];
  try { queues = (await api.request('/projects/' + p.id + '/queues')).items || []; }
  catch (e) { return emptyState('layers', 'No queues', ''); }
  if (!queues.length) return emptyState('layers', 'No queues', 'This project has no queues yet.');
  const results = await Promise.allSettled(queues.slice(0, 8).map((q) => api.request('/queues/' + q.id + '/stats')));
  const rows = queues.slice(0, 8).map((q, i) => {
    const r = results[i];
    const s = r && r.status === 'fulfilled' ? r.value : null;
    return '<tr><td class="cell-primary">' + esc(q.name) + '</td><td>' + badge(q.status) + '</td>' +
      '<td>' + (s ? num(s.queued) : '—') + '</td><td>' + (s ? num(s.running) : '—') + '</td>' +
      '<td>' + (s ? num(s.completed) : '—') + '</td><td>' + (s ? num(s.failed) : '—') + '</td></tr>';
  }).join('');
  return '<div class="table-scroll"><table><thead><tr><th>Queue</th><th>Status</th><th>Queued</th><th>Running</th><th>Completed</th><th>Failed</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
}

function donutChart(jobs, total) {
  const order = [['queued', '#3b82f6'], ['scheduled', '#8b5cf6'], ['running', '#22d3ee'], ['claimed', '#38bdf8'], ['completed', '#10b981'], ['failed', '#ef4444'], ['cancelled', '#71717a']];
  const legend = order.map(([k, c]) => '<div class="legend-item"><span class="legend-dot" style="background:' + c + '"></span>' + esc(k) + '<span class="legend-val">0</span></div>').join('');
  if (!total) {
    return '<div class="split-chart"><div class="donut" style="background:conic-gradient(#27272a 0 100%)"><div class="donut-center"><span class="big">0</span><span class="small">jobs</span></div></div><div class="legend">' + legend + '</div></div>';
  }
  let acc = 0;
  const stops = order.map(([k, c]) => {
    const v = num(jobs[k]);
    const from = acc / total * 100;
    acc += v;
    return { k, c, from, to: acc / total * 100, v };
  }).filter((s) => s.v > 0);
  const grad = stops.length ? 'conic-gradient(' + stops.map((s) => s.c + ' ' + s.from + '% ' + s.to + '%').join(',') + ')' : 'conic-gradient(#27272a 0 100%)';
  const leg = stops.map((s) => '<div class="legend-item"><span class="legend-dot" style="background:' + s.c + '"></span>' + esc(s.k) + '<span class="legend-val">' + s.v.toLocaleString() + '</span></div>').join('');
  return '<div class="split-chart"><div class="donut" style="background:' + grad + '"><div class="donut-center"><span class="big">' + total.toLocaleString() + '</span><span class="small">jobs</span></div></div><div class="legend">' + leg + '</div></div>';
}

function throughputBars(m) {
  const rows = [['Submitted', num(m.jobs_submitted), '#3b82f6'], ['Completed', num(m.jobs_completed), '#10b981'], ['Failed', num(m.jobs_failed), '#ef4444'], ['Retried', num(m.jobs_retried), '#f59e0b'], ['Dead-lettered', num(m.jobs_dead_lettered), '#ef4444']];
  const max = Math.max(1, ...rows.map((r) => r[1]));
  const success = num(m.jobs_completed), fail = num(m.jobs_failed);
  const successRate = (success + fail) ? Math.round(success / (success + fail) * 100) : 0;
  const bars = rows.map((r) => '<div class="bar-row"><span class="bar-label">' + r[0] + '</span><div class="bar-track"><div class="bar-fill" style="width:' + (r[1] / max * 100).toFixed(1) + '%;background:' + r[2] + '"></div></div><span class="bar-val">' + r[1].toLocaleString() + '</span></div>').join('');
  return '<div class="bars">' + bars + '</div>' +
    '<div class="two-col mt" style="gap:12px">' +
    '<div class="card stat"><div class="stat-top"><span class="stat-label">Success rate</span></div><div class="stat-value green">' + successRate + '%</div></div>' +
    '<div class="card stat"><div class="stat-top"><span class="stat-label">Avg execution</span></div><div class="stat-value">' + (m.avg_execution_ms ? m.avg_execution_ms.toFixed(1) + 'ms' : '—') + '</div></div>' +
    '</div>';
}

/* ---------------- queues ---------------- */
async function queuesView() {
  await ensureProjects();
  const p = state.currentProject;
  if (!p) return emptyState('layers', 'No projects', 'Create a project to start adding queues.');
  const data = await api.request('/projects/' + p.id + '/queues');
  const queues = data.items || [];
  const results = await Promise.allSettled(queues.map((q) => api.request('/queues/' + q.id + '/stats')));
  const statById = {};
  queues.forEach((q, i) => { const r = results[i]; statById[q.id] = (r && r.status === 'fulfilled') ? r.value : null; });

  const rows = queues.map((q) => {
    const s = statById[q.id];
    const counts = s ? '<span class="muted">q' + num(s.queued) + '</span> / <span class="muted">r' + num(s.running) + '</span> / <span class="muted">c' + num(s.completed) + '</span> / <span class="muted" style="color:var(--red)">f' + num(s.failed) + '</span>' : '—';
    return '<tr><td><div class="cell-primary">' + esc(q.name) + '</div><div class="cell-sub">' + esc(q.description || '') + '</div></td>' +
      '<td>' + num(q.priority) + '</td><td>' + num(q.concurrency) + '</td>' +
      '<td class="mono muted">' + (q.retry_policy_id ? esc(shortId(q.retry_policy_id)) : '—') + '</td>' +
      '<td>' + badge(q.status) + '</td><td>' + counts + '</td>' +
      '<td class="actions">' +
      (q.status === 'active' ? '<button class="icon-btn" title="Pause" onclick="pauseQueue(\'' + esc(q.id) + '\')">' + icon('pause') + '</button>' : '<button class="icon-btn" title="Resume" onclick="resumeQueue(\'' + esc(q.id) + '\')">' + icon('play') + '</button>') +
      '<button class="icon-btn" title="Configure" onclick="editQueue(\'' + esc(q.id) + '\')">' + icon('pencil') + '</button>' +
      '<button class="icon-btn danger" title="Delete" onclick="deleteQueue(\'' + esc(q.id) + '\',\'' + esc(q.name) + '\')">' + icon('trash') + '</button>' +
      '</td></tr>';
  }).join('');

  return '<div class="page-head"><div><div class="page-title">Queues</div><div class="page-sub">' + esc(p.name) + ' · ' + queues.length + ' queue' + (queues.length === 1 ? '' : 's') + '</div></div>' +
    '<button class="btn" onclick="openQueueDialog()">' + icon('plus') + ' New Queue</button></div>' +
    (queues.length
      ? '<div class="table-wrap"><div class="table-scroll"><table><thead><tr><th>Name</th><th>Priority</th><th>Concurrency</th><th>Retry policy</th><th>Status</th><th>Jobs (q/r/c/f)</th><th></th></tr></thead><tbody>' + rows + '</tbody></table></div></div>'
      : emptyState('layers', 'No queues', 'Create your first queue to start processing jobs.'));
}

/* ---------------- jobs ---------------- */
async function jobsView() {
  await ensureProjects();
  const p = state.currentProject;
  if (!p) return emptyState('list-ordered', 'No projects', 'Create a project to submit jobs.');
  const qs = new URLSearchParams();
  if (state.jobs.queue_id) qs.set('queue_id', state.jobs.queue_id);
  if (state.jobs.status) qs.set('status', state.jobs.status);
  if (state.jobs.type) qs.set('type', state.jobs.type);
  qs.set('limit', '25');
  if (state.jobs.cursor) qs.set('cursor', state.jobs.cursor);

  const [data, queueData] = await Promise.all([
    api.request('/projects/' + p.id + '/jobs?' + qs),
    api.request('/projects/' + p.id + '/queues')
  ]);
  const queues = queueData.items || [];
  const nameById = {};
  queues.forEach((q) => { nameById[q.id] = q.name; });

  const items = data.items || [];
  const rows = items.map((j) => '<tr class="clickable" onclick="showJob(\'' + esc(j.id) + '\')">' +
    '<td class="mono">' + esc(shortId(j.id)) + '</td><td>' + esc(j.type) + '</td><td>' + num(j.priority) + '</td><td>' + badge(j.status) + '</td>' +
    '<td>' + esc(nameById[j.queue_id] || shortId(j.queue_id) || '—') + '</td>' +
    '<td>' + num(j.attempts) + '/' + num(j.max_attempts) + '</td>' +
    '<td>' + (j.worker_id ? '<span class="mono muted">' + esc(shortId(j.worker_id)) + '</span>' : '—') + '</td>' +
    '<td class="cell-sub">' + fmtRel(j.created_at) + '</td>' +
    '<td class="cell-sub">' + (j.scheduled_at ? fmtDate(j.scheduled_at) : '—') + '</td>' +
    '<td class="muted wrap">' + (j.last_error ? esc(truncate(j.last_error, 40)) : '—') + '</td></tr>').join('');

  const statusOpts = ['queued', 'scheduled', 'claimed', 'running', 'completed', 'failed', 'cancelled']
    .map((s) => '<option value="' + s + '"' + (state.jobs.status === s ? ' selected' : '') + '>' + s + '</option>').join('');
  const typeOpts = ['immediate', 'delayed', 'scheduled', 'recurring']
    .map((t) => '<option value="' + t + '"' + (state.jobs.type === t ? ' selected' : '') + '>' + t + '</option>').join('');
  const queueOpts = '<option value="">All queues</option>' + queues.map((q) => '<option value="' + esc(q.id) + '"' + (state.jobs.queue_id === q.id ? ' selected' : '') + '>' + esc(q.name) + '</option>').join('');

  return '<div class="page-head"><div><div class="page-title">Jobs</div><div class="page-sub">' + esc(p.name) + ' · ' + items.length + ' shown</div></div>' +
    '<button class="btn" onclick="openJobDialog()">' + icon('plus') + ' Submit Job</button></div>' +
    '<div class="toolbar" style="margin-bottom:14px">' +
    '<select class="select" onchange="setJobFilter(\'queue_id\', this.value)">' + queueOpts + '</select>' +
    '<select class="select" onchange="setJobFilter(\'status\', this.value)"><option value="">All statuses</option>' + statusOpts + '</select>' +
    '<select class="select" onchange="setJobFilter(\'type\', this.value)"><option value="">All types</option>' + typeOpts + '</select>' +
    '<button class="btn btn-secondary btn-sm" onclick="clearJobFilters()">' + icon('x') + ' Clear</button></div>' +
    (items.length
      ? '<div class="table-wrap"><div class="table-scroll"><table><thead><tr><th>ID</th><th>Type</th><th>Priority</th><th>Status</th><th>Queue</th><th>Attempts</th><th>Worker</th><th>Created</th><th>Scheduled</th><th>Error</th></tr></thead><tbody>' + rows + '</tbody></table></div></div>' + jobsPager(items.length)
      : emptyState('list-ordered', 'No jobs', 'Jobs matching the current filters will appear here.'));
}

function jobsPager(shown) {
  return '<div class="pager">' +
    '<button class="btn btn-outline btn-sm" onclick="prevJobs()"' + (state.jobs.history.length ? '' : ' disabled') + '>' + icon('chevron-left') + ' Prev</button>' +
    '<span>' + shown + ' shown</span>' +
    '<button class="btn btn-outline btn-sm" onclick="nextJobs()"' + (state.jobs.cursor ? '' : ' disabled') + '>Next ' + icon('chevron-right') + '</button></div>';
}

/* ---------------- workers ---------------- */
async function workersView() {
  const data = await api.request('/workers');
  const items = data.items || [];
  const rows = items.map((w) => {
    const age = (Date.now() - new Date(w.last_heartbeat).getTime()) / 1000;
    const health = w.status === 'dead' ? 'dead' : age < 30 ? 'healthy' : age < 60 ? 'stale' : 'unhealthy';
    return '<tr><td><div class="mono">' + esc(w.id) + '</div><div class="cell-sub">' + esc(w.hostname || '—') + '</div></td>' +
      '<td>' + badge(w.status) + '</td><td>' + num(w.concurrency) + '</td>' +
      '<td class="cell-sub">' + fmtRel(w.last_heartbeat) + '</td><td>' + healthBadge(health) + '</td></tr>';
  }).join('');
  return '<div class="page-head"><div><div class="page-title">Workers</div><div class="page-sub">' + items.length + ' worker' + (items.length === 1 ? '' : 's') + ' registered</div></div></div>' +
    (items.length
      ? '<div class="table-wrap"><div class="table-scroll"><table><thead><tr><th>Worker</th><th>Status</th><th>Concurrency</th><th>Last heartbeat</th><th>Health</th></tr></thead><tbody>' + rows + '</tbody></table></div></div>'
      : emptyState('server', 'No workers', 'Workers that connect and send heartbeats will appear here.'));
}

function healthBadge(h) {
  const map = { healthy: 'active', stale: 'paused', unhealthy: 'failed', dead: 'dead' };
  const label = { healthy: 'Healthy', stale: 'Stale', unhealthy: 'Unhealthy', dead: 'Dead' };
  return '<span class="badge ' + map[h] + '">' + label[h] + '</span>';
}

/* ---------------- dead letter ---------------- */
async function dlqView() {
  await ensureProjects();
  const p = state.currentProject;
  if (!p) return emptyState('archive', 'No projects', 'Create a project to see dead-lettered jobs.');
  const qs = new URLSearchParams({ limit: '25' });
  if (state.dlq.cursor) qs.set('cursor', state.dlq.cursor);
  const data = await api.request('/projects/' + p.id + '/dead-letter?' + qs);
  const items = data.items || [];
  const rows = items.map((d) => '<tr>' +
    '<td class="mono">' + esc(shortId(d.job_id)) + '</td><td>' + num(d.attempts) + '</td>' +
    '<td class="wrap">' + esc(d.reason || '—') + '</td>' +
    '<td>' + (d.worker_id ? '<span class="mono muted">' + esc(shortId(d.worker_id)) + '</span>' : '—') + '</td>' +
    '<td class="cell-sub">' + fmtRel(d.failed_at) + '</td>' +
    '<td>' + (d.requeued_at ? '<span class="badge muted">requeued</span>' : '<button class="btn btn-outline btn-sm" onclick="requeue(\'' + esc(d.id) + '\')">' + icon('refresh-cw') + ' Requeue</button>') + '</td></tr>').join('');
  return '<div class="page-head"><div><div class="page-title">Dead Letter</div><div class="page-sub">' + esc(p.name) + ' · permanently failed jobs</div></div></div>' +
    (items.length
      ? '<div class="table-wrap"><div class="table-scroll"><table><thead><tr><th>Job</th><th>Attempts</th><th>Reason</th><th>Worker</th><th>Failed</th><th></th></tr></thead><tbody>' + rows + '</tbody></table></div></div>' + dlqPager(items.length)
      : emptyState('archive', 'No dead-lettered jobs', 'Jobs that exhaust all retry attempts will appear here.'));
}

function dlqPager(shown) {
  return '<div class="pager">' +
    '<button class="btn btn-outline btn-sm" onclick="prevDlq()"' + (state.dlq.history.length ? '' : ' disabled') + '>' + icon('chevron-left') + ' Prev</button>' +
    '<span>' + shown + ' shown</span>' +
    '<button class="btn btn-outline btn-sm" onclick="nextDlq()"' + (state.dlq.cursor ? '' : ' disabled') + '>Next ' + icon('chevron-right') + '</button></div>';
}

/* ---------------- metrics ---------------- */
async function metricsView() {
  const m = await api.request('/metrics');
  const submitted = num(m.jobs_submitted), completed = num(m.jobs_completed), failed = num(m.jobs_failed), retried = num(m.jobs_retried);
  const successRate = (completed + failed) ? Math.round(completed / (completed + failed) * 100) : 0;
  const retryRate = submitted ? Math.round(retried / submitted * 100) : 0;
  const cards = [
    ['Uptime', fmtUptime(m.uptime_seconds), ''],
    ['Jobs submitted', submitted.toLocaleString(), ''],
    ['Jobs completed', completed.toLocaleString(), 'green'],
    ['Jobs failed', failed.toLocaleString(), 'red'],
    ['Jobs retried', retried.toLocaleString(), 'amber'],
    ['Dead-lettered', num(m.jobs_dead_lettered).toLocaleString(), 'red'],
    ['Claims made', num(m.claims_made).toLocaleString(), ''],
    ['Leases recovered', num(m.leases_recovered).toLocaleString(), ''],
    ['Executions done', num(m.executions_done).toLocaleString(), ''],
    ['Avg execution', (m.avg_execution_ms ? m.avg_execution_ms.toFixed(1) + 'ms' : '—'), '']
  ];
  return '<div class="page-head"><div><div class="page-title">Metrics</div><div class="page-sub">Throughput counters since the server started.</div></div></div>' +
    '<div class="grid-cards">' + cards.map((c) => '<div class="card stat"><div class="stat-top"><span class="stat-label">' + esc(c[0]) + '</span></div><div class="stat-value ' + c[2] + '">' + esc(c[1]) + '</div></div>').join('') + '</div>' +
    '<div class="two-col mt">' +
    '<div class="card"><div class="card-head"><div class="card-title">Throughput</div></div>' + throughputBars(m) + '</div>' +
    '<div class="card"><div class="card-head"><div class="card-title">Rates</div></div><div class="bars">' +
    '<div class="bar-row"><span class="bar-label">Success rate</span><div class="bar-track"><div class="bar-fill" style="width:' + successRate + '%;background:#10b981"></div></div><span class="bar-val">' + successRate + '%</span></div>' +
    '<div class="bar-row"><span class="bar-label">Retry rate</span><div class="bar-track"><div class="bar-fill" style="width:' + Math.min(100, retryRate) + '%;background:#f59e0b"></div></div><span class="bar-val">' + retryRate + '%</span></div>' +
    '</div></div></div>';
}

/* ---------------- shared ---------------- */
function emptyState(iconName, title, desc) {
  return '<div class="empty">' + icon(iconName) + '<div class="t">' + esc(title) + '</div>' + (desc ? '<div class="d">' + esc(desc) + '</div>' : '') + '</div>';
}
function truncate(s, n) { s = String(s || ''); return s.length > n ? s.slice(0, n) + '…' : s; }

function prettyPayload(p) {
  if (p == null) return '{}';
  if (typeof p === 'string') {
    try { const obj = JSON.parse(p); return JSON.stringify(obj, null, 2); } catch (e) {}
    try { const obj = JSON.parse(atob(p)); return JSON.stringify(obj, null, 2); } catch (e) {}
    return p;
  }
  try { return JSON.stringify(p, null, 2); } catch (e) { return String(p); }
}

async function ensureProjects() {
  if (state.projects.length) return;
  const data = await api.request('/projects?limit=50');
  state.projects = data.items || [];
  if (!state.currentProject && state.projects.length) state.currentProject = state.projects[0];
  refreshProjectPicker();
}
function refreshProjectPicker() {
  const sel = $('#projectPicker');
  if (!sel) return;
  const cur = state.currentProject ? state.currentProject.id : '';
  sel.innerHTML = state.projects.map((p) => '<option value="' + esc(p.id) + '"' + (p.id === cur ? ' selected' : '') + '>' + esc(p.name) + '</option>').join('');
}
function switchProject(id) {
  state.currentProject = state.projects.find((p) => p.id === id) || null;
  state.jobs.cursor = ''; state.jobs.history = [];
  state.dlq.cursor = ''; state.dlq.history = [];
  render();
}

/* ---------------- actions ---------------- */
async function pauseQueue(id) { try { await api.request('/queues/' + id + '/pause', { method: 'POST' }); toast('Queue paused', 'ok'); } catch (e) { toast(e.message, 'error'); } render(); }
async function resumeQueue(id) { try { await api.request('/queues/' + id + '/resume', { method: 'POST' }); toast('Queue resumed', 'ok'); } catch (e) { toast(e.message, 'error'); } render(); }
async function deleteQueue(id, name) {
  if (!confirm('Delete queue "' + name + '"? This cascades to its jobs.')) return;
  try { await api.request('/queues/' + id, { method: 'DELETE' }); toast('Queue deleted', 'ok'); } catch (e) { toast(e.message, 'error'); }
  render();
}
async function retryJob(id) { try { await api.request('/jobs/' + id + '/retry', { method: 'POST' }); toast('Job re-queued', 'ok'); } catch (e) { toast(e.message, 'error'); } closeDrawer(); render(); }
async function cancelJob(id) { try { await api.request('/jobs/' + id + '/cancel', { method: 'POST' }); toast('Job cancelled', 'ok'); } catch (e) { toast(e.message, 'error'); } closeDrawer(); render(); }
async function requeue(id) { try { await api.request('/dead-letter/' + id + '/requeue', { method: 'POST' }); toast('Job requeued', 'ok'); } catch (e) { toast(e.message, 'error'); } render(); }

function setJobFilter(k, v) { state.jobs[k] = v; state.jobs.cursor = ''; state.jobs.history = []; render(); }
function clearJobFilters() { state.jobs = { queue_id: '', status: '', type: '', cursor: '', history: [] }; render(); }

function nextJobs() { if (!state.jobs.cursor) return; state.jobs.history.push(state.jobs.cursor); render(); }
function prevJobs() { if (!state.jobs.history.length) return; state.jobs.cursor = state.jobs.history.pop(); render(); }
function nextDlq() { if (!state.dlq.cursor) return; state.dlq.history.push(state.dlq.cursor); render(); }
function prevDlq() { if (!state.dlq.history.length) return; state.dlq.cursor = state.dlq.history.pop(); render(); }

/* ---------------- drawer ---------------- */
function openDrawer(html) { const r = $('#drawer-root'); if (r) r.innerHTML = '<div class="drawer-overlay" onclick="closeDrawer()"></div><div class="drawer">' + html + '</div>'; }
function closeDrawer() { const r = $('#drawer-root'); if (r) r.innerHTML = ''; }

async function showJob(id) {
  openDrawer('<div class="drawer-head">' + icon('list-ordered') + '<div style="flex:1"><div class="card-title">Job details</div><div class="mono muted">' + esc(shortId(id)) + '</div></div><button class="icon-btn" onclick="closeDrawer()">' + icon('x') + '</button></div><div class="drawer-body"><div class="skeleton skel-row"></div></div>');
  try {
    const [job, execs, logs] = await Promise.all([
      api.request('/jobs/' + id),
      api.request('/jobs/' + id + '/executions'),
      api.request('/jobs/' + id + '/logs?limit=100')
    ]);
    const ex = execs.items || [];
    const lg = logs.items || [];
    const exRows = ex.map((e) => '<tr><td>' + num(e.attempt) + '</td><td>' + badge(e.status) + '</td><td class="mono muted">' + (e.worker_id ? esc(shortId(e.worker_id)) : '—') + '</td><td>' + fmtDur(e.duration_ms) + '</td><td class="muted wrap">' + esc(e.error || '') + '</td></tr>').join('');
    const logLines = lg.map((l) => '<div class="log-line"><span class="mono muted">' + fmtTime(l.created_at) + '</span> <span class="badge ' + (l.level === 'error' ? 'failed' : 'muted') + '" style="padding:0 6px">' + esc(l.level) + '</span> <span>' + esc(l.message) + '</span></div>').join('');

    const body = $('#drawer-root .drawer-body');
    if (!body) return;
    body.innerHTML = '<div class="kv">' +
      '<div class="k">Status</div><div class="v">' + badge(job.status) + '</div>' +
      '<div class="k">Type</div><div class="v">' + esc(job.type) + '</div>' +
      '<div class="k">Priority</div><div class="v">' + num(job.priority) + '</div>' +
      '<div class="k">Attempts</div><div class="v">' + num(job.attempts) + ' / ' + num(job.max_attempts) + '</div>' +
      '<div class="k">Retry strategy</div><div class="v">' + esc(job.retry_strategy || '—') + '</div>' +
      '<div class="k">Worker</div><div class="v mono">' + (job.worker_id ? esc(job.worker_id) : '—') + '</div>' +
      '<div class="k">Created</div><div class="v">' + fmtTime(job.created_at) + '</div>' +
      '<div class="k">Scheduled</div><div class="v">' + (job.scheduled_at ? fmtTime(job.scheduled_at) : '—') + '</div>' +
      '<div class="k">Started</div><div class="v">' + (job.started_at ? fmtTime(job.started_at) : '—') + '</div>' +
      '<div class="k">Completed</div><div class="v">' + (job.completed_at ? fmtTime(job.completed_at) : '—') + '</div>' +
      '<div class="k">Failed</div><div class="v">' + (job.failed_at ? fmtTime(job.failed_at) : '—') + '</div>' +
      '</div>' +
      (job.last_error ? '<div class="error-box">' + icon('alert-triangle') + '<div>' + esc(job.last_error) + '</div></div>' : '') +
      '<div><div class="card-title" style="margin-bottom:8px">Payload</div><pre class="payload">' + esc(prettyPayload(job.payload)) + '</pre></div>' +
      '<div><div class="card-title" style="margin-bottom:8px">Executions</div>' + (ex.length ? '<div class="table-scroll"><table><thead><tr><th>Attempt</th><th>Status</th><th>Worker</th><th>Duration</th><th>Error</th></tr></thead><tbody>' + exRows + '</tbody></table></div>' : emptyState('activity', 'No executions', '')) + '</div>' +
      '<div><div class="card-title" style="margin-bottom:8px">Logs</div>' + (lg.length ? '<div class="log-lines">' + logLines + '</div>' : emptyState('activity', 'No logs', '')) + '</div>';

    const drawer = $('#drawer-root .drawer');
    if (drawer) {
      drawer.insertAdjacentHTML('beforeend', '<div class="drawer-foot">' +
        (job.status === 'failed' ? '<button class="btn btn-secondary" onclick="retryJob(\'' + esc(job.id) + '\')">' + icon('refresh-cw') + ' Retry</button>' : '') +
        (['completed', 'failed', 'cancelled'].indexOf(job.status) === -1 ? '<button class="btn btn-destructive" onclick="cancelJob(\'' + esc(job.id) + '\')">' + icon('ban') + ' Cancel</button>' : '') +
        '</div>');
    }
  } catch (e) {
    const body = $('#drawer-root .drawer-body');
    if (body) body.innerHTML = '<div class="error-box">' + icon('alert-triangle') + '<div>' + esc(e.message) + '</div></div>';
  }
}

/* ---------------- dialogs ---------------- */
function openDialog(html) { const r = $('#dialog-root'); if (r) r.innerHTML = '<div class="overlay" onclick="if(event.target===this)closeDialog()">' + html + '</div>'; }
function closeDialog() { const r = $('#dialog-root'); if (r) r.innerHTML = ''; }

function dialogShell(title, body) {
  return '<div class="dialog"><div class="dialog-head"><div class="dialog-title">' + esc(title) + '</div><button class="icon-btn" onclick="closeDialog()">' + icon('x') + '</button></div>' +
    '<form id="dlgForm"><div class="dialog-body">' + body + '</div><div class="dialog-foot"><button type="button" class="btn btn-ghost" onclick="closeDialog()">Cancel</button><button type="submit" class="btn">Save</button></div></form></div>';
}

function queueForm(q) {
  return '<div class="form-grid">' +
    '<div class="field full"><label>Name</label><input class="input" name="name" required value="' + esc(q ? q.name : '') + '" placeholder="default" /></div>' +
    '<div class="field full"><label>Description</label><input class="input" name="description" value="' + esc(q ? q.description : '') + '" placeholder="Optional" /></div>' +
    '<div class="field"><label>Priority</label><input class="input" name="priority" type="number" value="' + (q ? num(q.priority) : 0) + '" /></div>' +
    '<div class="field"><label>Concurrency</label><input class="input" name="concurrency" type="number" min="1" value="' + (q ? num(q.concurrency) : 1) + '" /></div>' +
    '<div class="field"><label>Retry strategy</label><select class="select" name="retry_strategy">' +
    ['fixed', 'linear', 'exponential'].map((s) => '<option value="' + s + '">' + s + '</option>').join('') + '</select></div>' +
    '<div class="field"><label>Max attempts</label><input class="input" name="max_attempts" type="number" min="0" value="3" /></div>' +
    '<div class="field"><label>Initial delay (ms)</label><input class="input" name="initial_delay_ms" type="number" min="0" value="1000" /></div>' +
    '<div class="field"><label>Max delay (ms)</label><input class="input" name="max_delay_ms" type="number" min="0" value="60000" /></div>' +
    '<div class="field"><label>Multiplier</label><input class="input" name="multiplier" type="number" step="0.1" min="1" value="2" /></div>' +
    '</div>';
}

function openQueueDialog(q) {
  openDialog(dialogShell(q ? 'Configure queue' : 'New queue', queueForm(q)));
  const form = $('#dlgForm');
  if (form) form.addEventListener('submit', (e) => { e.preventDefault(); submitQueueForm(q); });
}
function editQueue(id) {
  api.request('/queues/' + id).then((q) => openQueueDialog(q)).catch((e) => toast(e.message, 'error'));
}
async function submitQueueForm(q) {
  const form = $('#dlgForm');
  if (!form) return;
  const fd = new FormData(form);
  const body = {
    name: fd.get('name') || '',
    description: fd.get('description') || '',
    priority: parseInt(fd.get('priority') || '0', 10) || 0,
    concurrency: parseInt(fd.get('concurrency') || '1', 10) || 1,
    retry_strategy: fd.get('retry_strategy') || 'exponential',
    max_attempts: parseInt(fd.get('max_attempts') || '3', 10) || 0,
    initial_delay_ms: parseInt(fd.get('initial_delay_ms') || '1000', 10) || 0,
    max_delay_ms: parseInt(fd.get('max_delay_ms') || '60000', 10) || 0,
    multiplier: parseFloat(fd.get('multiplier') || '2') || 1
  };
  try {
    if (q) await api.request('/queues/' + q.id, { method: 'PATCH', body: JSON.stringify(body) });
    else await api.request('/projects/' + state.currentProject.id + '/queues', { method: 'POST', body: JSON.stringify(body) });
    toast(q ? 'Queue updated' : 'Queue created', 'ok');
    closeDialog();
    render();
  } catch (e) { toast(e.message, 'error'); }
}

function openJobDialog() {
  openDialog(dialogShell('Submit job', jobForm()));
  const form = $('#dlgForm');
  if (form) form.addEventListener('submit', (e) => { e.preventDefault(); submitJobForm(); });
}
function jobForm() {
  return '<div class="form-grid">' +
    '<div class="field full"><label>Queue</label><select class="select" name="queue_id" id="jobQueueSelect"></select></div>' +
    '<div class="field"><label>Type</label><select class="select" name="type"><option value="immediate">immediate</option><option value="delayed">delayed</option><option value="scheduled">scheduled</option><option value="recurring">recurring</option></select></div>' +
    '<div class="field"><label>Priority</label><input class="input" name="priority" type="number" value="0" /></div>' +
    '<div class="field full"><label>Payload (JSON)</label><textarea class="input" name="payload">{}</textarea></div>' +
    '<div class="field"><label>Delay (ms)</label><input class="input" name="delay_ms" type="number" min="0" value="0" /></div>' +
    '<div class="field"><label>Scheduled at</label><input class="input" name="scheduled_at" type="datetime-local" /></div>' +
    '<div class="field full"><label>Cron expression</label><input class="input" name="cron_expr" placeholder="*/5 * * * *" /></div>' +
    '</div>';
}
async function submitJobForm() {
  const form = $('#dlgForm');
  if (!form) return;
  const fd = new FormData(form);
  const type = fd.get('type') || 'immediate';
  let payload = {};
  try { payload = JSON.parse(fd.get('payload') || '{}'); } catch (e) { toast('Payload must be valid JSON', 'error'); return; }
  const body = { type: type, payload: payload, priority: parseInt(fd.get('priority') || '0', 10) || 0 };
  if (type === 'delayed') body.delay_ms = parseInt(fd.get('delay_ms') || '0', 10) || 0;
  if (type === 'scheduled') { const v = fd.get('scheduled_at'); if (v) body.scheduled_at = new Date(v).toISOString(); else { toast('scheduled_at is required for scheduled jobs', 'error'); return; } }
  if (type === 'recurring') body.cron_expr = fd.get('cron_expr') || '';
  const queueId = fd.get('queue_id');
  if (!queueId) { toast('Select a queue', 'error'); return; }
  try {
    await api.request('/projects/' + state.currentProject.id + '/queues/' + queueId + '/jobs', { method: 'POST', body: JSON.stringify(body) });
    toast(type === 'recurring' ? 'Schedule created' : 'Job submitted', 'ok');
    closeDialog();
    render();
  } catch (e) { toast(e.message, 'error'); }
}
async function populateJobQueues() {
  if (!state.currentProject) return;
  const sel = $('#jobQueueSelect');
  if (!sel) return;
  try {
    const data = await api.request('/projects/' + state.currentProject.id + '/queues');
    const queues = data.items || [];
    sel.innerHTML = queues.map((q) => '<option value="' + esc(q.id) + '">' + esc(q.name) + '</option>').join('');
  } catch (e) {}
}

function openProjectDialog() {
  openDialog(dialogShell('New project', '<div class="field"><label>Name</label><input class="input" name="name" required placeholder="My Project" /></div><div class="field"><label>Description</label><input class="input" name="description" placeholder="Optional" /></div>'));
  const form = $('#dlgForm');
  if (form) form.addEventListener('submit', (e) => { e.preventDefault(); submitProjectForm(); });
}
async function submitProjectForm() {
  const form = $('#dlgForm');
  if (!form) return;
  const fd = new FormData(form);
  try {
    await api.request('/projects', { method: 'POST', body: JSON.stringify({ name: fd.get('name') || '', description: fd.get('description') || '' }) });
    state.projects = [];
    state.currentProject = null;
    toast('Project created', 'ok');
    closeDialog();
    await ensureProjects();
    render();
  } catch (e) { toast(e.message, 'error'); }
}

/* ---------------- routes + boot ---------------- */
const routes = { overview, queues: queuesView, jobs: jobsView, workers: workersView, 'dead-letter': dlqView, metrics: metricsView };

function init() {
  applyIcons(document);
  const keyInput = $('#apiKey');
  if (keyInput && api.key) keyInput.value = api.key;

  const connect = $('#connectBtn');
  if (connect) connect.addEventListener('click', () => {
    api.key = (keyInput ? keyInput.value : '').trim();
    localStorage.setItem('scheduler_api_key', api.key);
    state.projects = []; state.currentProject = null;
    toast('Connected', 'ok');
    render();
  });

  const picker = $('#projectPicker');
  if (picker) picker.addEventListener('change', (e) => switchProject(e.target.value));

  const newProjectBtn = $('#newProjectBtn');
  if (newProjectBtn) newProjectBtn.addEventListener('click', openProjectDialog);

  const menuBtn = $('#menuBtn'), closeBtn = $('#sidebarClose'), scrim = $('#scrim'), sidebar = $('#sidebar');
  if (menuBtn) menuBtn.addEventListener('click', () => { if (sidebar) sidebar.classList.add('open'); if (scrim) scrim.classList.add('show'); });
  if (closeBtn) closeBtn.addEventListener('click', () => { if (sidebar) sidebar.classList.remove('open'); if (scrim) scrim.classList.remove('show'); });
  if (scrim) scrim.addEventListener('click', () => { if (sidebar) sidebar.classList.remove('open'); scrim.classList.remove('show'); });

  window.addEventListener('hashchange', render);

  // Populate queue selector when the job dialog opens.
  const dialogObserver = new MutationObserver(() => { if ($('#jobQueueSelect')) populateJobQueues(); });
  dialogObserver.observe(document.getElementById('dialog-root'), { childList: true, subtree: true });

  if (!location.hash) location.hash = '#/overview';
  render();
  setInterval(() => {
    if (!$('#dialog-root .overlay')) render();
  }, 3000);
}

document.addEventListener('DOMContentLoaded', init);

const $ = s => document.querySelector(s), $$ = s => document.querySelectorAll(s);
let current = null, progress = null, createRequestID = null;
const rid = () => crypto.randomUUID();
const iso = value => new Date(value).toISOString();
const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const localTime = value => value ? new Date(value).toLocaleString() : '—';

async function api(path, options = {}) {
  const response = await fetch(path, {headers:{'Content-Type':'application/json'}, ...options});
  const data = await response.json();
  if (!response.ok) {
    if (response.status === 409) $('#conflict').hidden = false;
    throw new Error(data.detail || '请求失败');
  }
  return data;
}
function toast(message) {
  const box = $('#toast');
  box.textContent = message;
  box.classList.add('show');
  setTimeout(() => box.classList.remove('show'), 2500);
}
function payload(form) { return Object.fromEntries(new FormData(form).entries()); }
function env() { return {request_id:rid(), expected_revision:current.revision}; }

async function loadList() {
  const data = await api('/api/cases');
  $('#case-list').innerHTML = data.cases.map(c => `<button class="case-item" data-id="${esc(c.case_id)}"><strong>${esc(c.case_id)} ${c.related_cases?.length ? '↔' : ''}</strong><span>${esc(c.room_code)} · ${esc(c.status)} · r${c.revision}</span></button>`).join('') || '<p>尚无案件</p>';
  $$('.case-item').forEach(button => button.onclick = () => loadCase(button.dataset.id));
}
async function loadCase(id) {
  current = await api(`/api/cases/${encodeURIComponent(id)}`);
  progress = await api(`/api/cases/${encodeURIComponent(id)}/retests/progress`);
  $('#empty').hidden = true;
  $('#case-view').hidden = false;
  $('#conflict').hidden = true;
  render();
  await loadPreflight();
}
function render() {
  const c = current;
  $('#case-status').textContent = c.status;
  $('#case-title').textContent = `${c.case_id} · ${c.room_code}`;
  $('#case-meta').textContent = `${c.excursion_kind} / ${c.sample_point} / 观测 ${c.observed_value}，限值 ${c.limit_value} / 修订 ${c.revision}`;
  $('#case-related').innerHTML = c.related_cases?.length ? `<span class="overlap-mark">存在交叠范围</span> ${c.related_cases.map(x => `<button class="link-button" data-related="${esc(x.case_id)}">${esc(x.case_id)} · ${esc(x.status)} · ${localTime(x.overlap_start)} 至 ${localTime(x.overlap_end)}</button>`).join('')}` : '';
  $$('[data-related]').forEach(button => button.onclick = () => loadCase(button.dataset.related));
  $('#timeline').innerHTML = [...c.timeline].reverse().map(e => `<div class="timeline-item"><strong>${esc(e.type)}</strong><p>${esc(e.summary)}</p><small>r${e.revision} · ${esc(e.actor_id)} · ${localTime(e.at)}</small></div>`).join('');
  renderActions();
  renderRetests();
  renderArchive();
  const frozen = c.status === 'released';
  ['#investigation-form','#action-form','#retest-form','#review-form'].forEach(selector => $(selector).querySelectorAll('input,textarea,select,button').forEach(control => control.disabled = frozen));
}
function renderActions() {
  const actions = current.corrective_actions;
  $('#actions').innerHTML = actions.map(a => {
    const original = a.completed_at ? `<small>原完成：${localTime(a.completed_at)} · 证据 ${esc(a.evidence_digest)}</small>` : '';
    const revoked = a.status === 'revoked' ? `<small class="failure">撤销：${esc(a.revocation_reason)} · ${esc(a.revoked_by)} · ${localTime(a.revoked_at)}</small>` : '';
    const replacement = a.replaced_action_id ? `<small>替代措施：${esc(a.replaced_action_id)}</small>` : '';
    const controls = current.status === 'correcting' ? `${a.status === 'open' ? `<button data-complete="${esc(a.action_id)}">登记完成</button>` : ''}${a.status !== 'revoked' ? `<button data-revoke="${esc(a.action_id)}">撤销</button>` : ''}` : '';
    return `<div class="action"><span><strong>${esc(a.action_id)} · ${esc(a.description)}</strong><br>${esc(a.owner_id)} · ${esc(a.status)}<br>${original}${revoked}${replacement}</span><span>${controls}</span></div>`;
  }).join('') || '<p>尚无纠正措施</p>';
  $$('[data-complete]').forEach(button => button.onclick = () => completeAction(button.dataset.complete));
  $$('[data-revoke]').forEach(button => button.onclick = () => revokeAction(button.dataset.revoke));
  const used = new Set(actions.map(a => a.replaced_action_id).filter(Boolean));
  const options = actions.filter(a => a.status === 'revoked' && !used.has(a.action_id));
  $('#action-form [name=replaced_action_id]').innerHTML = '<option value="">非替代措施</option>' + options.map(a => `<option value="${esc(a.action_id)}">替代 ${esc(a.action_id)}：${esc(a.description)}</option>`).join('');
}
function renderRetests() {
  const p = progress;
  $('#retest-form [name=limit_value]').value = p.applicable_limit;
  const boundary = p.latest_failure_boundary ? ` · 最近失败边界 ${localTime(p.latest_failure_boundary)}` : '';
  $('#retest-progress').innerHTML = `<div class="progress"><strong>${p.eligible_for_review ? '已达到待复核资格' : `连续合格 ${p.consecutive_passing}/${p.required_passing_rounds}，还差 ${p.remaining_rounds} 轮`}</strong>${boundary}</div>${p.blocking_reasons.map(x => `<div class="check failure">${esc(x)}</div>`).join('')}`;
  $('#retests').innerHTML = p.segments.map(segment => `<section class="segment"><h4>序列段 ${segment.number}</h4>${segment.rounds.map(x => {
    const delta = x.limit_value - x.observed_value;
    return `<div class="retest-row"><span>第 ${x.sequence} 轮 · ${localTime(x.sampled_at)} · 读数 ${x.observed_value} / 阈值 ${x.limit_value} · 阈值差 ${delta}</span><strong>${x.outcome === 'pass' ? '合格，计入连续轮次' : '失败，终止本段'}</strong></div>`;
  }).join('') || '<p>等待本段首轮复测</p>'}</section>`).join('');
}
async function loadPreflight() {
  if (!current) return;
  const p = await api(`/api/cases/${current.case_id}/review/preflight`);
  $('#preflight').innerHTML = `<p><strong>${p.eligible ? '门禁已满足' : '尚有阻断项'}</strong> · 连续合格 ${p.consecutive_passing}/${p.required_passing_rounds}</p>${p.blocking_reasons.map(x => `<div class="check">${esc(x)}</div>`).join('')}`;
}
async function submit(path, body) {
  try {
    const data = await api(path, {method:'POST', body:JSON.stringify(body)});
    current = data.case;
    progress = data.retest_progress;
    render();
    await loadList();
    await loadPreflight();
    toast(data.replayed ? '已返回首次操作结果' : '操作已保存');
  } catch (error) { toast(error.message); }
}

async function previewOverlaps() {
  const form = $('#create-form'), values = payload(form), box = $('#overlap-preview'), confirm = $('#overlap-confirm-wrap');
  if (!values.room_code || !values.affected_window_start || !values.affected_window_end) {
    box.textContent = '填写房间和影响时间窗后自动检查关联案件。';
    confirm.hidden = true;
    return [];
  }
  try {
    const query = new URLSearchParams({room_code:values.room_code, affected_window_start:iso(values.affected_window_start), affected_window_end:iso(values.affected_window_end)});
    const data = await api(`/api/cases/overlaps?${query}`);
    confirm.hidden = !data.confirmation_required;
    if (!data.confirmation_required) {
      box.innerHTML = '<strong>未发现同房间交叠案件。</strong>';
      return [];
    }
    box.innerHTML = `<strong>发现 ${data.overlaps.length} 个闭区间交叠案件：</strong>${data.overlaps.map(x => `<div>${esc(x.case_id)} · ${esc(x.status)} · ${esc(x.excursion_kind)} · ${localTime(x.overlap_start)} 至 ${localTime(x.overlap_end)}</div>`).join('')}`;
    return data.overlaps;
  } catch (error) {
    box.textContent = error.message;
    confirm.hidden = true;
    return null;
  }
}
$('#new-case').onclick = () => { createRequestID = rid(); $('#create-dialog').showModal(); };
$('#cancel-create').onclick = () => $('#create-dialog').close();
$('#reload').onclick = () => loadCase(current.case_id);
['room_code','affected_window_start','affected_window_end'].forEach(name => $(
  `#create-form [name=${name}]`
).addEventListener('change', previewOverlaps));
$('#create-form').onsubmit = async event => {
  event.preventDefault();
  const overlaps = await previewOverlaps();
  if (overlaps === null) return;
  const values = payload(event.target);
  if (overlaps.length && !values.overlap_confirmed) { toast('请确认已查看关联影响范围'); return; }
  ['observed_value','limit_value'].forEach(k => values[k] = Number(values[k]));
  ['occurred_at','affected_window_start','affected_window_end'].forEach(k => values[k] = iso(values[k]));
  values.overlap_confirmed = Boolean(values.overlap_confirmed);
  values.request_id = createRequestID;
  try {
    const data = await api('/api/cases', {method:'POST', body:JSON.stringify(values)});
    $('#create-dialog').close();
    current = data.case;
    progress = data.retest_progress;
    await loadList();
    await loadCase(current.case_id);
    event.target.reset();
    createRequestID = null;
    toast('案件已登记并冻结影响范围');
  } catch (error) { toast(error.message); }
};
$('#investigation-form').onsubmit = event => {
  event.preventDefault();
  const values = payload(event.target);
  submit(`/api/cases/${current.case_id}/investigation`, {...values,...env(),investigation_id:rid(),evidence_digests:values.evidence_digests.split(',').map(x=>x.trim()).filter(Boolean)});
};
$('#action-form').onsubmit = event => {
  event.preventDefault();
  const values = payload(event.target);
  submit(`/api/cases/${current.case_id}/actions`, {...values,...env(),action_id:rid(),actor_id:values.owner_id});
};
async function completeAction(id) {
  const evidence = prompt('输入该措施的证据摘要');
  if (evidence) submit(`/api/cases/${current.case_id}/actions/${id}/complete`, {...env(),evidence_digest:evidence,actor_id:'operator-ui'});
}
async function revokeAction(id) {
  const reason = prompt('输入撤销理由');
  if (!reason) return;
  const actor = prompt('输入撤销操作人');
  if (actor) submit(`/api/cases/${current.case_id}/actions/${id}/revoke`, {...env(),reason,actor_id:actor});
}
$('#retest-form').onsubmit = event => {
  event.preventDefault();
  const values = {...payload(event.target),...env(),round_id:rid()};
  values.sampled_at = iso(values.sampled_at);
  values.observed_value = Number(values.observed_value);
  values.limit_value = Number(values.limit_value);
  submit(`/api/cases/${current.case_id}/retests`, values);
};
$('#review-form').onsubmit = event => {
  event.preventDefault();
  submit(`/api/cases/${current.case_id}/review`, {...payload(event.target),...env()});
};

function verificationHTML(record) {
  if (!record) return '<p>尚无人工校验记录。</p>';
  const result = record.result;
  return `<div class="verification ${result.valid ? '' : 'failed'}"><p><strong>${result.valid ? '校验通过' : '校验失败'}</strong> · ${esc(record.verifier_id)} · ${localTime(record.executed_at)}</p>${result.checks.map(c => `<div class="check ${c.ok ? '' : 'failure'}"><span>${esc(c.name)} · ${esc(c.location)}<br><small>期望：${esc(c.expected)}<br>实际：${esc(c.actual)}</small></span><strong>${c.ok?'✓':'✗'}</strong></div>`).join('')}</div>`;
}
async function renderArchive() {
  const box = $('#archive');
  if (!current?.archive) { box.innerHTML=''; return; }
  const caseId = current.case_id;
  box.innerHTML='<p>正在读取放行摘要…</p>';
  try {
    const [summary, audits] = await Promise.all([api(`/api/cases/${caseId}/archive/summary`), api(`/api/cases/${caseId}/archive/verify`)]);
    if (current?.case_id !== caseId) return;
    box.innerHTML = `<h3>恢复放行档案</h3><p><strong>${esc(summary.excursion_label)}</strong> · ${esc(summary.room_code)} · ${esc(summary.excursion_reading)}</p><p>根因：${esc(summary.root_cause)}</p><p>${summary.action_count} 项纠正措施 · ${summary.passing_retests} 轮合格复测 · ${esc(summary.approved_by)} 于 ${localTime(summary.approved_at)} 批准</p><p><code>${esc(summary.canonical_digest)}</code></p><button id="verify">重新计算摘要并留痕</button><div id="verify-result">${verificationHTML(audits.latest)}</div><h4>校验历史（最新在前）</h4><div id="verify-history">${audits.history.map(verificationHTML).join('')}</div>`;
    $('#verify').onclick = async () => {
      const verifier = prompt('输入本次校验人');
      if (!verifier) return;
      try {
        await api(`/api/cases/${caseId}/archive/verify`, {method:'POST',body:JSON.stringify({request_id:rid(),verifier_id:verifier})});
        await renderArchive();
      } catch(error) { toast(error.message); }
    };
  } catch (error) { box.innerHTML = `<p class="alert">${esc(error.message)}</p>`; }
}
$$('nav button').forEach(button => button.onclick = () => {
  $$('nav button').forEach(x => x.classList.toggle('active',x===button));
  $$('.tab').forEach(x => x.hidden=x.id!==button.dataset.tab);
});
api('/api/health').then(() => {
  $('#connection').textContent='服务正常';
  loadList();
}).catch(() => $('#connection').textContent='连接失败');

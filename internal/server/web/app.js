const sky = document.getElementById('sky');
const ctx = sky.getContext('2d');
const statusEl = document.getElementById('status');
const epochEl = document.getElementById('epoch');
const snrBody = document.getElementById('snr-body');
const snrTable = document.getElementById('snr-table');
const countsEl = document.getElementById('counts');
const portEl = document.getElementById('port');
const errorEl = document.getElementById('error');
const skyLegendEl = document.getElementById('sky-legend');
const tableFilterEl = document.getElementById('table-filter');
const showSlipCountsEl = document.getElementById('show-slip-counts');

const BAND_LABELS = { l1: 'L1', l2: 'L2', l5: 'L5', l6: 'L6' };

/** Band columns — auto-hidden when empty across visible rows. */
const BAND_COLS = ['l1', 'l2', 'l5', 'l6'];

const SLIP_RANK = { none: 0, '300': 1, '60': 2, now: 3 };
const SLIP_RING = { now: '#f85149', '60': '#d29922', '300': '#8b6914' };

/** Sky plot color key — click to filter by constellation (hover for description). */
const SKY_LEGEND = [
  { key: 'GPS', label: 'GPS', color: '#2563eb', description: 'United States Global Positioning System.', match: ['GPS'] },
  { key: 'SBAS', label: 'SBAS', color: '#22c55e', description: 'Satellite-Based Augmentation System (WAAS, EGNOS, MSAS, GAGAN).', match: ['SBAS'] },
  { key: 'GLONASS', label: 'GLONASS', color: '#ef4444', description: 'Russian GLONASS constellation.', match: ['GLONASS'] },
  { key: 'Galileo', label: 'Galileo', color: '#a16207', description: 'European Galileo GNSS.', match: ['Galileo'] },
  { key: 'Beidou', label: 'BeiDou', color: '#9333ea', description: 'Chinese BeiDou Navigation Satellite System (BDS).', match: ['Beidou'] },
  { key: 'MSS', label: 'MSS', color: '#0ea5e9', description: 'Mobile Satellite Service (OmniSTAR, Terralite).', match: ['OmniSTAR', 'Terralite'] },
];

const OTHER_LEGEND = {
  key: 'other',
  label: 'Other',
  color: '#8b949e',
  description: 'Other GNSS systems (QZSS, IRNSS, etc.).',
  match: ['QZSS', 'IRNSS'],
};

const ARROW_UP = '\u25B2';
const ARROW_DOWN = '\u25BC';

let allRows = [];
let sortCol = 'sv';
let sortDir = 1;
let filterLegendKey = null;
let lastEpochSec = null;
let showSlipCounts = localStorage.getItem('showSlipCounts') === '1';

/** @type {Map<string, {ignoreNextSlip: boolean}>} */
const svSlipState = new Map();
/** @type {Map<string, {lastSlipEpochSec: number|null, prevCount: number, slipCount: number}>} */
const sigSlipState = new Map();

function rad(deg) { return (deg * Math.PI) / 180; }

function positiveAz(az) {
  let a = az % 360;
  if (a < 0) a += 360;
  return a;
}

function epochSec(week, timeSec) {
  return week * 604800 + timeSec;
}

function svKey(row) {
  return `${row.system}-${row.svid}`;
}

function sigKey(row, sig) {
  return `${svKey(row)}-${sig.blockType}-${sig.trackType}`;
}

function maxSlipLevel(a, b) {
  return SLIP_RANK[a] >= SLIP_RANK[b] ? a : b;
}

function slipLevelFromAge(lastSlipEpochSec, currentEpochSec) {
  if (lastSlipEpochSec == null) return 'none';
  const age = currentEpochSec - lastSlipEpochSec;
  if (age > 300) return 'none';
  if (age <= 60) return '60';
  return '300';
}

function slipDetected(sig, st) {
  const count = sig.cycleSlipCount ?? 0;
  if (sig.cycleSlipNow) return true;
  return count > st.prevCount;
}

function slipLevelLabel(level) {
  if (level === 'now') return 'this epoch';
  if (level === '60') return '< 60 s';
  if (level === '300') return '< 300 s';
  return 'none';
}

function enrichSlipState(rt27, rows) {
  const current = epochSec(rt27.week, rt27.timeSec);
  if (lastEpochSec != null && current < lastEpochSec - 1) {
    svSlipState.clear();
    sigSlipState.clear();
  }
  lastEpochSec = current;

  for (const row of rows) {
    const sk = svKey(row);
    if (!svSlipState.has(sk)) {
      svSlipState.set(sk, { ignoreNextSlip: true });
    }
    const sv = svSlipState.get(sk);
    row._slipLevel = 'none';

    for (const col of BAND_COLS) {
      const band = row[col];
      if (!band?.signals?.length) {
        if (band) band._slipLevel = 'none';
        continue;
      }
      let bandLevel = 'none';
      for (const sig of band.signals) {
        const key = sigKey(row, sig);
        if (!sigSlipState.has(key)) {
          sigSlipState.set(key, {
            lastSlipEpochSec: null,
            prevCount: sig.cycleSlipCount ?? 0,
            slipCount: 0,
          });
        }
        const st = sigSlipState.get(key);
        let level = slipLevelFromAge(st.lastSlipEpochSec, current);

        if (slipDetected(sig, st)) {
          if (sv.ignoreNextSlip) {
            sv.ignoreNextSlip = false;
          } else {
            st.lastSlipEpochSec = current;
            st.slipCount++;
            level = 'now';
          }
        }
        st.prevCount = sig.cycleSlipCount ?? st.prevCount;
        sig._slipLevel = level;
        sig._slipCount = st.slipCount;
        bandLevel = maxSlipLevel(bandLevel, level);
      }
      band._slipLevel = bandLevel;
      row._slipLevel = maxSlipLevel(row._slipLevel, bandLevel);
    }
  }
}

function signalHoverDetail(sig) {
  const parts = [
    sig.trackName || 'signal',
    `SNR ${sig.snr.toFixed(1)} dB-Hz`,
  ];
  if (sig._slipLevel && sig._slipLevel !== 'none') {
    parts.push(`cycle slip ${slipLevelLabel(sig._slipLevel)}`);
  }
  parts.push(`slip count ${sig._slipCount ?? 0}`);
  if (sig.cycleSlipCount != null && sig.cycleSlipCount > 0) {
    parts.push(`RX counter ${sig.cycleSlipCount}`);
  }
  if (sig.trackHint) parts.push(sig.trackHint);
  return parts.join(' · ');
}

function bandHoverText(band, col) {
  const label = BAND_LABELS[col] || col.toUpperCase();
  if (!band?.signals?.length) return `${label}: no data`;
  return `${label}: ${band.signals.map(signalHoverDetail).join(' · ')}`;
}

function escapeAttr(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;');
}

function fmtTd(col, cls, content, title) {
  const t = title ? ` title="${escapeAttr(title)}"` : '';
  return `<td class="${cls}" data-col="${col}"${t}>${content}</td>`;
}

function legendEntryForKey(key) {
  if (key === OTHER_LEGEND.key) return OTHER_LEGEND;
  return SKY_LEGEND.find(i => i.key === key) || null;
}

function slipCellClass(level) {
  if (level && level !== 'none') return ` slip-${level}`;
  return '';
}

function legendForSystem(name) {
  for (const item of SKY_LEGEND) {
    if (item.match.includes(name)) return item;
  }
  if (OTHER_LEGEND.match.includes(name)) return OTHER_LEGEND;
  return null;
}

function systemColor(name, sys) {
  const item = legendForSystem(name);
  if (item) return item.color;
  return `hsl(${(sys * 47) % 360}, 60%, 55%)`;
}

function rowMatchesFilter(row) {
  if (!filterLegendKey) return true;
  const item = legendEntryForKey(filterLegendKey);
  if (!item) return true;
  return item.match.includes(row.systemName);
}

function visibleRows() {
  return allRows.filter(rowMatchesFilter);
}

function initSkyLegend() {
  skyLegendEl.innerHTML = '';
  for (const item of SKY_LEGEND) {
    skyLegendEl.appendChild(makeLegendButton(item));
  }
}

function makeLegendButton(item) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'legend-item';
  btn.dataset.key = item.key;
  btn.title = `${item.description} Click to show only ${item.label}. Click again to show all.`;
  btn.innerHTML = `<span class="legend-swatch" style="background:${item.color}"></span><span>${item.label}</span>`;
  btn.addEventListener('click', () => toggleConstellationFilter(item.key));
  return btn;
}

function toggleConstellationFilter(key) {
  filterLegendKey = filterLegendKey === key ? null : key;
  updateLegendUI();
  refreshDataView();
}

function clearConstellationFilter() {
  filterLegendKey = null;
  updateLegendUI();
  refreshDataView();
}

function updateLegendUI() {
  skyLegendEl.querySelectorAll('.legend-item').forEach(el => {
    const key = el.dataset.key;
    const isActive = filterLegendKey === key;
    el.classList.toggle('filter-active', isActive);
    el.classList.toggle('dimmed', filterLegendKey && !isActive);
  });

  const item = filterLegendKey ? legendEntryForKey(filterLegendKey) : null;
  if (item) {
    const n = visibleRows().length;
    tableFilterEl.hidden = false;
    tableFilterEl.innerHTML = `Filtered to <strong>${escapeHtml(item.label)}</strong> (${n} SV${n === 1 ? '' : 's'}).<button type="button" id="clear-filter">Show all</button>`;
    document.getElementById('clear-filter')?.addEventListener('click', clearConstellationFilter);
  } else {
    tableFilterEl.hidden = true;
    tableFilterEl.innerHTML = '';
  }
}

function bandHasData(band) {
  return band && band.present && band.signals && band.signals.length > 0;
}

function updateEmptyColumns(rows) {
  const empty = new Set();
  for (const col of BAND_COLS) {
    if (!rows.some(row => bandHasData(row[col]))) empty.add(col);
  }

  snrTable.querySelectorAll('thead th, tbody td').forEach(el => {
    const col = el.dataset.col;
    if (!col || !BAND_COLS.includes(col)) return;
    el.classList.toggle('col-empty', empty.has(col));
  });
}

function fmtDMS(deg, isLat) {
  const hemi = isLat ? (deg >= 0 ? 'N' : 'S') : (deg >= 0 ? 'E' : 'W');
  const abs = Math.abs(deg);
  const d = Math.floor(abs);
  const minFull = (abs - d) * 60;
  const m = Math.floor(minFull);
  const s = (minFull - m) * 60;
  const mPad = isLat ? String(m).padStart(2, '0') : String(m);
  return `${d}\u00b0 ${mPad}' ${s.toFixed(5)}" ${hemi}`;
}

function drawSky(rows) {
  const w = sky.width;
  const h = sky.height;
  const cx = w / 2;
  const cy = h / 2;
  const r = Math.min(cx, cy) - 24;

  ctx.clearRect(0, 0, w, h);
  ctx.fillStyle = '#121820';
  ctx.fillRect(0, 0, w, h);

  ctx.strokeStyle = '#30363d';
  ctx.lineWidth = 1;
  for (const el of [90, 60, 30, 0]) {
    const rr = r * (1 - el / 90);
    ctx.beginPath();
    ctx.arc(cx, cy, rr, 0, Math.PI * 2);
    ctx.stroke();
  }
  for (let az = 0; az < 360; az += 45) {
    const a = rad(az - 90);
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.lineTo(cx + r * Math.cos(a), cy + r * Math.sin(a));
    ctx.stroke();
  }

  ctx.fillStyle = '#8b949e';
  ctx.font = '11px system-ui';
  ctx.textAlign = 'center';
  ctx.fillText('N', cx, cy - r - 8);
  ctx.fillText('S', cx, cy + r + 14);
  ctx.fillText('E', cx + r + 10, cy + 4);
  ctx.fillText('W', cx - r - 10, cy + 4);

  if (!rows?.length) return;

  for (const row of rows) {
    const el = row.elevation ?? 0;
    const az = positiveAz(row.azimuth ?? 0);
    const rr = r * (1 - el / 90);
    const a = rad(az - 90);
    const x = cx + rr * Math.cos(a);
    const y = cy + rr * Math.sin(a);
    const color = systemColor(row.systemName, row.system);
    const slipLevel = row._slipLevel || 'none';

    if (slipLevel !== 'none') {
      ctx.beginPath();
      ctx.arc(x, y, 11, 0, Math.PI * 2);
      ctx.strokeStyle = SLIP_RING[slipLevel];
      ctx.lineWidth = slipLevel === 'now' ? 3 : 2;
      ctx.stroke();
    }

    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(x, y, 7, 0, Math.PI * 2);
    ctx.fill();

    ctx.fillStyle = '#e6edf3';
    ctx.font = '10px system-ui';
    ctx.textAlign = 'center';
    ctx.fillText(String(row.svid), x, y + 3);
  }
}

function updatePosition(pos, rt27) {
  const set = (id, val) => { document.getElementById(id).textContent = val; };
  if (!pos) {
    ['lat', 'lon', 'alt', 'hdop', 'rms', 'svs', 'aug'].forEach(id => set(id, '—'));
  } else {
    set('lat', fmtDMS(pos.latitude, true));
    set('lon', fmtDMS(pos.longitude, false));
    set('alt', pos.altitude.toFixed(2) + ' m');
    set('hdop', pos.hdop.toFixed(2));
    set('rms', pos.rms.toFixed(4) + ' m');
    set('svs', `${pos.svsUsed} / ${pos.svsTracked}`);
    set('aug', pos.augmentationText || String(pos.augmentation));
  }
  set('antenna', rt27?.antennas || '—');
}

function bandSNR(band) {
  if (!band?.signals?.length) return -1;
  return Math.max(...band.signals.map(s => s.snr));
}

function compareRows(a, b, col) {
  switch (col) {
    case 'system':
      return a.systemName.localeCompare(b.systemName) || a.svid - b.svid;
    case 'sv':
      return a.svid - b.svid || a.system - b.system;
    case 'elevation':
      return a.elevation - b.elevation || a.svid - b.svid;
    case 'azimuth':
      return positiveAz(a.azimuth) - positiveAz(b.azimuth) || a.svid - b.svid;
    case 'l2': return bandSNR(a.l2) - bandSNR(b.l2);
    case 'l5': return bandSNR(a.l5) - bandSNR(b.l5);
    case 'l6': return bandSNR(a.l6) - bandSNR(b.l6);
    case 'l1': return bandSNR(a.l1) - bandSNR(b.l1);
    default:
      return a.system - b.system || a.svid - b.svid;
  }
}

function sortedRows(rows) {
  if (!rows) return [];
  return [...rows].sort((a, b) => sortDir * compareRows(a, b, sortCol));
}

function updateSortHeaders() {
  document.querySelectorAll('#snr-table th.sortable').forEach(th => {
    th.classList.remove('sorted-asc', 'sorted-desc');
    const label = th.dataset.label || th.textContent.replace(/[\u25B2\u25BC]/g, '').trim();
    th.dataset.label = label;
    if (th.dataset.sort === sortCol) {
      th.classList.add(sortDir === 1 ? 'sorted-asc' : 'sorted-desc');
      th.textContent = `${label} ${sortDir === 1 ? ARROW_UP : ARROW_DOWN}`;
    } else {
      th.textContent = label;
    }
  });
}

function fmtBandCell(band, col) {
  const slipCls = slipCellClass(band?._slipLevel);
  const cls = `band col-${col}${slipCls}`;
  const title = bandHoverText(band, col);
  if (!band?.present || !band.signals?.length) {
    return fmtTd(col, `${cls} empty`, '—', title);
  }
  const snrs = band.signals.map(s => s.snr.toFixed(1)).join('/');
  const tracks = band.signals.map(s => escapeHtml(s.trackName || '')).join('/');
  let slipLine = '';
  if (showSlipCounts) {
    const counts = band.signals.map(s => String(s._slipCount ?? 0)).join('/');
    slipLine = `<span class="slip-val">${counts}</span>`;
  }
  const body = `<span class="snr-val">${snrs}</span><span class="track-val">${tracks}</span>${slipLine}`;
  return fmtTd(col, cls, body, title);
}

function updateSNRTable(rows) {
  snrBody.innerHTML = '';
  updateSortHeaders();
  updateEmptyColumns(rows);
  if (!rows.length) return;

  for (const row of sortedRows(rows)) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      ${fmtTd('system', 'col-system', escapeHtml(row.systemName), `${row.systemName} · system ${row.system}`)}
      ${fmtTd('sv', 'col-sv', String(row.svid), `${row.systemName} PRN ${row.svid}`)}
      ${fmtTd('elevation', 'col-elevation', String(row.elevation), `Elevation ${row.elevation}°`)}
      ${fmtTd('azimuth', 'col-azimuth', String(positiveAz(row.azimuth)), `Azimuth ${positiveAz(row.azimuth)}°`)}
      ${fmtBandCell(row.l1, 'l1')}
      ${fmtBandCell(row.l2, 'l2')}
      ${fmtBandCell(row.l5, 'l5')}
      ${fmtBandCell(row.l6, 'l6')}`;
    snrBody.appendChild(tr);
  }
  updateEmptyColumns(rows);
}

function refreshDataView() {
  const rows = visibleRows();
  drawSky(rows);
  updateSNRTable(rows);
  updateLegendUI();
}

function escapeHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;');
}

function applySnapshot(snap) {
  statusEl.textContent = snap.connected ? 'Connected' : 'Disconnected';
  statusEl.className = 'status ' + (snap.connected ? 'connected' : 'disconnected');

  countsEl.textContent = `packets: ${snap.packetCount} · raw: ${snap.rawCount}`;
  portEl.textContent = snap.port ? `port: ${snap.port}` : 'demo mode';
  errorEl.textContent = snap.lastError || '';

  if (snap.rt27) {
    allRows = snap.rt27.svs || [];
    enrichSlipState(snap.rt27, allRows);
    const t = snap.rt27.timeSec.toFixed(3);
    const total = allRows.length;
    const shown = visibleRows().length;
    let epoch = `GPS week ${snap.rt27.week} · t=${t}s · ${snap.rt27.numSVs} SVs · ${total} rows`;
    if (filterLegendKey && shown !== total) epoch += ` · showing ${shown}`;
    epochEl.textContent = epoch;
    refreshDataView();
  }

  updatePosition(snap.position, snap.rt27);
}

document.querySelectorAll('#snr-table th.sortable').forEach(th => {
  th.dataset.label = th.textContent.trim();
  th.addEventListener('click', () => {
    const col = th.dataset.sort;
    if (sortCol === col) sortDir = -sortDir;
    else {
      sortCol = col;
      sortDir = (col === 'l1' || col === 'l2' || col === 'l5' || col === 'l6') ? -1 : 1;
    }
    updateSNRTable(visibleRows());
  });
});

function connectSSE() {
  const es = new EventSource('/api/events');
  es.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === 'snapshot' && msg.snapshot) applySnapshot(msg.snapshot);
    } catch (_) { /* ignore */ }
  };
  es.onerror = () => {
    es.close();
    setTimeout(connectSSE, 2000);
  };
}

fetch('/api/snapshot').then(r => r.json()).then(applySnapshot).catch(() => {});

connectSSE();
initSkyLegend();
drawSky([]);
updateSortHeaders();

if (showSlipCountsEl) {
  showSlipCountsEl.checked = showSlipCounts;
  showSlipCountsEl.addEventListener('change', () => {
    showSlipCounts = showSlipCountsEl.checked;
    localStorage.setItem('showSlipCounts', showSlipCounts ? '1' : '0');
    updateSNRTable(visibleRows());
  });
}

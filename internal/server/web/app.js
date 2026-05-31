const sky = document.getElementById('sky');
const ctx = sky.getContext('2d');
const statusEl = document.getElementById('status');
const epochEl = document.getElementById('epoch');
const snrBody = document.getElementById('snr-body');
const snrTable = document.getElementById('snr-table');
const countsEl = document.getElementById('counts');
const portEl = document.getElementById('port');
const errorEl = document.getElementById('error');
const connectFormEl = document.getElementById('connect-form');
const connectHostEl = document.getElementById('connect-host');
const connectPortEl = document.getElementById('connect-port');
const connectDisconnectEl = document.getElementById('connect-disconnect');
const connectErrorEl = document.getElementById('connect-error');
const skyLegendEl = document.getElementById('sky-legend');
const tableFilterEl = document.getElementById('table-filter');
const showSlipCountsEl = document.getElementById('show-slip-counts');
const showTrackTypesEl = document.getElementById('show-track-types');
const showTrackTypesWrapEl = document.getElementById('show-track-types-wrap');
const themeSelectEl = document.getElementById('theme-select');
const slipMaskEl = document.getElementById('slip-mask');
const filterUsedEl = document.getElementById('filter-used');
const antennaFilterEl = document.getElementById('antenna-filter');
const singleAntLegendEl = document.getElementById('single-ant-legend');

const BAND_LABELS = { l1: 'L1', l2: 'L2', l5: 'L5', l6: 'L6' };

const DEFAULT_CONNECT_HOST = 'sps855.com';
const DEFAULT_CONNECT_PORT = '28005';
const CONNECT_HOST_KEY = 'connectHost';
const CONNECT_PORT_KEY = 'connectPort';

/** Band columns — auto-hidden when empty across visible rows. */
const BAND_COLS = ['l1', 'l2', 'l5', 'l6'];

const SLIP_RANK = { none: 0, '300': 1, '60': 2, now: 3 };
let slipRing = { now: '#f85149', '60': '#d29922', '300': '#8b6914' };
let usedRing = '#3fb950';
let singleAntRing = '#f0883e';

/** Sky plot color key — click to filter by constellation (hover for description). */
const SKY_LEGEND = [
  { key: 'GPS', label: 'GPS', color: '#2563eb', description: 'United States Global Positioning System.', match: ['GPS'] },
  { key: 'SBAS', label: 'SBAS', color: '#22c55e', description: 'Satellite-Based Augmentation System (WAAS, EGNOS, MSAS, GAGAN).', match: ['SBAS'] },
  { key: 'GLONASS', label: 'GLONASS', color: '#ef4444', description: 'Russian GLONASS constellation.', match: ['GLONASS'] },
  { key: 'Galileo', label: 'Galileo', color: '#a16207', description: 'European Galileo GNSS.', match: ['Galileo'] },
  { key: 'Beidou', label: 'BeiDou', color: '#9333ea', description: 'Chinese BeiDou Navigation Satellite System (BDS).', match: ['Beidou'] },
  { key: 'MSS', label: 'MSS', color: '#0ea5e9', description: 'Mobile Satellite Service (OmniSTAR, Terralite).', match: ['MSS'] },
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

const SLIP_MASK_KEY = 'slipMaskEl';
const SHOW_TRACK_TYPES_KEY = 'showTrackTypes';
const THEME_KEY = 'theme';
const DEFAULT_SLIP_MASK = 11;
const ANTENNA_FILTER_KEY = 'filterAntenna';

function cssVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function syncThemeColors() {
  slipRing = {
    now: cssVar('--slip-now-ring') || slipRing.now,
    '60': cssVar('--slip-60-ring') || slipRing['60'],
    '300': cssVar('--slip-300-ring') || slipRing['300'],
  };
  usedRing = cssVar('--ok') || usedRing;
  singleAntRing = cssVar('--single-ant-ring') || singleAntRing;
}

function applyTheme(mode) {
  themeMode = mode;
  localStorage.setItem(THEME_KEY, mode);
  const root = document.documentElement;
  if (mode === 'system') root.removeAttribute('data-theme');
  else root.setAttribute('data-theme', mode);
  syncThemeColors();
  refreshDataView();
}

function applyShowDev(show) {
  if (showTrackTypesWrapEl) showTrackTypesWrapEl.hidden = !show;
}

let allRows = [];
let sortCol = 'sv';
let sortDir = 1;
/** @type {Map<string, 'only' | 'hide'>} */
const filterSystemModes = new Map();
/** @type {null | 'used' | 'unused'} */
let filterUsedMode = null;
let hasPositionSVData = false;
let dualAntenna = false;
/** @type {'both' | '0' | '1'} */
let filterAntennaMode = localStorage.getItem(ANTENNA_FILTER_KEY) || 'both';
let serverConfig = { hosted: false, allowLocalHosts: false, showDev: false };
let themeMode = localStorage.getItem(THEME_KEY) || 'system';
let lastEpochSec = null;
let showSlipCounts = localStorage.getItem('showSlipCounts') === '1';
let showTrackTypes = localStorage.getItem(SHOW_TRACK_TYPES_KEY) === '1';

function clampSlipMask(value) {
  const n = Number.parseInt(String(value), 10);
  if (!Number.isFinite(n)) return DEFAULT_SLIP_MASK;
  return Math.min(90, Math.max(0, n));
}

function loadSlipMask() {
  const stored = localStorage.getItem(SLIP_MASK_KEY);
  if (stored == null) return DEFAULT_SLIP_MASK;
  return clampSlipMask(stored);
}

let slipMask = loadSlipMask();

function slipIndicated(row) {
  return (row?.elevation ?? 0) >= slipMask;
}

function displaySlipLevel(level, row) {
  if (!slipIndicated(row)) return 'none';
  return level && level !== 'none' ? level : 'none';
}

function applyConnectionFields(host, port) {
  if (host) connectHostEl.value = host;
  if (port != null && port !== '') connectPortEl.value = String(port);
}

function persistConnectionDraft(host, port) {
  const h = (host ?? connectHostEl?.value ?? '').trim();
  const p = port ?? connectPortEl?.value ?? '';
  if (h) localStorage.setItem(CONNECT_HOST_KEY, h);
  if (p !== '') localStorage.setItem(CONNECT_PORT_KEY, String(p));
}

function restoreConnectionFields(cfg) {
  const host = cfg?.lastHost || localStorage.getItem(CONNECT_HOST_KEY) || DEFAULT_CONNECT_HOST;
  const port = cfg?.lastPort || localStorage.getItem(CONNECT_PORT_KEY) || DEFAULT_CONNECT_PORT;
  applyConnectionFields(host, port);
  persistConnectionDraft(host, port);
}

applyConnectionFields(
  localStorage.getItem(CONNECT_HOST_KEY),
  localStorage.getItem(CONNECT_PORT_KEY),
);

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

function svIdentity(row) {
  return `${row.system}:${row.svid}`;
}

function svKey(row) {
  return `${row.system}-${row.svid}-${row.antenna ?? 0}`;
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

function signalHoverDetail(sig, row) {
  const parts = [
    sig.trackName || 'signal',
    `SNR ${sig.snr.toFixed(1)} dB-Hz`,
  ];
  if (serverConfig.showDev && showTrackTypes && sig.trackType != null) {
    parts.push(`track type ${sig.trackType}`);
  }
  if (slipIndicated(row) && sig._slipLevel && sig._slipLevel !== 'none') {
    parts.push(`cycle slip ${slipLevelLabel(sig._slipLevel)}`);
  }
  if (slipIndicated(row)) {
    parts.push(`slip count ${sig._slipCount ?? 0}`);
  }
  if (sig.cycleSlipCount != null && sig.cycleSlipCount > 0) {
    parts.push(`RX counter ${sig.cycleSlipCount}`);
  }
  if (sig.trackHint) parts.push(sig.trackHint);
  return parts.join(' · ');
}

function bandHoverText(band, col, row) {
  const label = BAND_LABELS[col] || col.toUpperCase();
  if (!band?.signals?.length) return `${label}: no data`;
  return `${label}: ${band.signals.map(sig => signalHoverDetail(sig, row)).join(' · ')}`;
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

function legendKeyForRow(row) {
  const item = legendForSystem(row.systemName);
  return item?.key ?? OTHER_LEGEND.key;
}

function rowMatchesSystemFilter(row) {
  const key = legendKeyForRow(row);
  const mode = filterSystemModes.get(key);
  if (mode === 'hide') return false;

  const onlyKeys = [];
  for (const [k, m] of filterSystemModes) {
    if (m === 'only') onlyKeys.push(k);
  }
  if (onlyKeys.length && !onlyKeys.includes(key)) return false;
  return true;
}

function rowMatchesAntennaFilter(row) {
  if (!dualAntenna || filterAntennaMode === 'both') return true;
  return String(row.antenna ?? 0) === filterAntennaMode;
}

function rowMatchesFilter(row) {
  if (!rowMatchesSystemFilter(row)) return false;
  if (!rowMatchesAntennaFilter(row)) return false;
  if (filterUsedMode === 'used' && !row.usedInSolution) return false;
  if (filterUsedMode === 'unused' && row.usedInSolution) return false;
  return true;
}

function hasActiveFilters() {
  const antennaFiltered = dualAntenna && filterAntennaMode !== 'both';
  return filterSystemModes.size > 0 || filterUsedMode != null || antennaFiltered;
}

function buildAntennaPresence(rows) {
  const map = new Map();
  for (const row of rows) {
    const k = svIdentity(row);
    if (!map.has(k)) map.set(k, new Set());
    map.get(k).add(row.antenna ?? 0);
  }
  return map;
}

function skyDisplayRows(rows) {
  if (!dualAntenna || filterAntennaMode !== 'both') return rows;
  const presence = buildAntennaPresence(allRows);
  const seen = new Map();
  for (const row of rows) {
    const k = svIdentity(row);
    const ants = presence.get(k);
    if (!seen.has(k)) {
      seen.set(k, row);
      row._singleAntennaOnly = ants != null && ants.size === 1;
      row._trackedAntennas = ants ? [...ants].sort((a, b) => a - b) : [row.antenna ?? 0];
    } else {
      const existing = seen.get(k);
      existing._slipLevel = maxSlipLevel(existing._slipLevel || 'none', row._slipLevel || 'none');
    }
  }
  return [...seen.values()];
}

function updateDualAntennaUI() {
  if (antennaFilterEl) antennaFilterEl.hidden = !dualAntenna;
  if (singleAntLegendEl) singleAntLegendEl.hidden = !dualAntenna || filterAntennaMode !== 'both';
  if (antennaFilterEl) {
    antennaFilterEl.querySelectorAll('.antenna-filter-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.ant === filterAntennaMode);
    });
  }
  updateAntennaColumnVisibility();
  document.querySelectorAll('.antenna-detail').forEach(el => {
    el.hidden = dualAntenna;
  });
}

function updateAntennaColumnVisibility() {
  snrTable.querySelectorAll('[data-col="antenna"]').forEach(el => {
    el.classList.toggle('col-empty', !dualAntenna);
  });
}

function setAntennaFilter(mode) {
  filterAntennaMode = mode;
  localStorage.setItem(ANTENNA_FILTER_KEY, mode);
  updateDualAntennaUI();
  refreshDataView();
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
  btn.title = `${item.description} Click: ${item.label} only · hide ${item.label} · show all.`;
  btn.innerHTML = `<span class="legend-swatch" style="background:${item.color}"></span><span>${item.label}</span>`;
  btn.addEventListener('click', () => toggleConstellationFilter(item.key));
  return btn;
}

function toggleConstellationFilter(key) {
  const current = filterSystemModes.get(key);
  if (!current) filterSystemModes.set(key, 'only');
  else if (current === 'only') filterSystemModes.set(key, 'hide');
  else filterSystemModes.delete(key);
  updateLegendUI();
  refreshDataView();
}

function toggleUsedFilter() {
  if (filterUsedMode === null) filterUsedMode = 'used';
  else if (filterUsedMode === 'used') filterUsedMode = 'unused';
  else filterUsedMode = null;
  updateLegendUI();
  refreshDataView();
}

function clearAllFilters() {
  filterSystemModes.clear();
  filterUsedMode = null;
  if (dualAntenna) {
    filterAntennaMode = 'both';
    localStorage.setItem(ANTENNA_FILTER_KEY, 'both');
  }
  updateLegendUI();
  refreshDataView();
}

function clearConstellationFilter() {
  clearAllFilters();
}

function usedFilterLabel(mode) {
  if (mode === 'used') return 'Used in position';
  if (mode === 'unused') return 'Not used in position';
  return 'Used in position';
}

function constellationFilterTitle(key, mode) {
  const item = legendEntryForKey(key);
  const label = item?.label ?? key;
  const desc = item?.description ?? '';
  if (mode === 'only') return `${desc} Showing ${label} only. Click to hide ${label}.`;
  if (mode === 'hide') return `${desc} Hiding ${label}. Click to show all systems.`;
  return `${desc} Click: ${label} only · hide ${label} · show all.`;
}

function updateLegendUI() {
  const anyOnly = [...filterSystemModes.values()].some(m => m === 'only');
  skyLegendEl.querySelectorAll('.legend-item[data-key]').forEach(el => {
    const key = el.dataset.key;
    const mode = filterSystemModes.get(key);
    el.classList.toggle('filter-active', mode === 'only');
    el.classList.toggle('filter-hidden', mode === 'hide');
    el.classList.toggle('dimmed', anyOnly && mode !== 'only');
    el.title = constellationFilterTitle(key, mode);
  });

  if (filterUsedEl) {
    filterUsedEl.classList.toggle('filter-active', filterUsedMode != null);
    filterUsedEl.classList.toggle('filter-unused', filterUsedMode === 'unused');
    filterUsedEl.disabled = !hasPositionSVData;
    filterUsedEl.classList.toggle('disabled', !hasPositionSVData);
    const label = filterUsedEl.querySelector('.used-filter-label');
    if (label) label.textContent = usedFilterLabel(filterUsedMode);
    filterUsedEl.title = filterUsedMode === null
      ? 'Click: used only · Click again: not used · Click again: show all'
      : filterUsedMode === 'used'
        ? 'Showing used in position only. Click for not used.'
        : 'Showing not used in position only. Click to show all.';
  }

  const filters = [];
  for (const [key, mode] of filterSystemModes) {
    const item = legendEntryForKey(key);
    if (!item) continue;
    if (mode === 'only') filters.push(`Only ${escapeHtml(item.label)}`);
    if (mode === 'hide') filters.push(`Hide ${escapeHtml(item.label)}`);
  }
  if (filterUsedMode === 'used') filters.push('Used in position');
  if (filterUsedMode === 'unused') filters.push('Not used in position');
  if (dualAntenna && filterAntennaMode !== 'both') {
    filters.push(`Antenna ${filterAntennaMode}`);
  }

  if (filters.length) {
    const n = visibleRows().length;
    tableFilterEl.hidden = false;
    tableFilterEl.innerHTML = `Filtered to ${filters.map(f => `<strong>${f}</strong>`).join(' + ')} (${n} SV${n === 1 ? '' : 's'}).<button type="button" id="clear-filter">Show all</button>`;
    document.getElementById('clear-filter')?.addEventListener('click', clearAllFilters);
  } else {
    tableFilterEl.hidden = true;
    tableFilterEl.innerHTML = '';
  }
}

function bandHasData(band) {
  return band && band.present && band.signals && band.signals.length > 0;
}

function rowHasNonL1Band(row) {
  return bandHasData(row.l2) || bandHasData(row.l5) || bandHasData(row.l6);
}

function fmtTrackLabel(sig) {
  const name = sig.trackName || '';
  if (!serverConfig.showDev || !showTrackTypes) return escapeHtml(name);
  return escapeHtml(`${name} (${sig.trackType ?? 0})`);
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
  ctx.fillStyle = cssVar('--sky-bg') || '#121820';
  ctx.fillRect(0, 0, w, h);

  ctx.strokeStyle = cssVar('--sky-grid') || '#30363d';
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

  ctx.fillStyle = cssVar('--sky-label') || '#8b949e';
  ctx.font = '11px system-ui';
  ctx.textAlign = 'center';
  ctx.fillText('N', cx, cy - r - 8);
  ctx.fillText('S', cx, cy + r + 14);
  ctx.fillText('E', cx + r + 10, cy + 4);
  ctx.fillText('W', cx - r - 10, cy + 4);

  if (!rows?.length) return;

  const plotRows = skyDisplayRows(rows);

  for (const row of plotRows) {
    const el = row.elevation ?? 0;
    const az = positiveAz(row.azimuth ?? 0);
    const rr = r * (1 - el / 90);
    const a = rad(az - 90);
    const x = cx + rr * Math.cos(a);
    const y = cy + rr * Math.sin(a);
    const color = systemColor(row.systemName, row.system);
    const slipLevel = displaySlipLevel(row._slipLevel, row);

    if (slipLevel !== 'none') {
      ctx.beginPath();
      ctx.arc(x, y, 11, 0, Math.PI * 2);
      ctx.strokeStyle = slipRing[slipLevel];
      ctx.lineWidth = slipLevel === 'now' ? 3 : 2;
      ctx.stroke();
    }

    if (row._singleAntennaOnly) {
      ctx.beginPath();
      ctx.arc(x, y, 13, 0, Math.PI * 2);
      ctx.strokeStyle = singleAntRing;
      ctx.lineWidth = 2;
      ctx.setLineDash([4, 3]);
      ctx.stroke();
      ctx.setLineDash([]);
    }

    if (hasPositionSVData && row.usedInSolution) {
      ctx.beginPath();
      ctx.arc(x, y, 9, 0, Math.PI * 2);
      ctx.strokeStyle = usedRing;
      ctx.lineWidth = 2;
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

function fmtM(value, digits, unit) {
  if (value == null || Number.isNaN(value)) return '—';
  return `${value.toFixed(digits)}${unit ? ` ${unit}` : ''}`;
}

function fmtVel(value) {
  return fmtM(value, 3, 'm/s');
}

function fmtSigma(value) {
  return fmtM(value, 4, 'm');
}

function fmtAge(pos) {
  if (pos?.rtk?.ageSec != null) return `${pos.rtk.ageSec.toFixed(2)} s`;
  return '—';
}

function svHoverDetail(row) {
  const parts = [`${row.systemName} PRN ${row.svid}`];
  if (dualAntenna) parts.push(`antenna ${row.antenna ?? 0}`);
  if (row._singleAntennaOnly && row._trackedAntennas?.length) {
    parts.push(`one antenna only (ant ${row._trackedAntennas[0]})`);
  }
  if (row.usedInSolution) parts.push('used in position');
  else parts.push('not used in position');
  if (row.raimFault) parts.push('RAIM fault');
  if (row.unhealthy) parts.push('unhealthy');
  return parts.join(' · ');
}

function fmtSVCell(row) {
  let tags = '';
  if (row.raimFault) {
    tags += '<span class="sv-tag sv-raim" title="RAIM fault">!</span>';
  }
  const body = tags
    ? `<span class="sv-cell"><span class="sv-id">${row.svid}</span>${tags}</span>`
    : String(row.svid);
  return fmtTd('sv', 'col-sv', body, svHoverDetail(row));
}

function fmtUsedCell(row) {
  if (!hasPositionSVData) {
    return fmtTd('used', 'col-used empty', '—', 'Position SV list not available');
  }
  const used = !!row.usedInSolution;
  const label = used ? 'Yes' : 'No';
  const cls = used ? 'col-used yes' : 'col-used no';
  const title = used ? 'Used in position solution' : 'Not used in position solution';
  return fmtTd('used', cls, label, title);
}

function updatePosition(pos, rt27) {
  const set = (id, val) => { document.getElementById(id).textContent = val; };
  const clearIds = [
    'lat', 'lon', 'alt', 'sigma-h', 'sigma-u', 'data-age', 'svs', 'aug',
    'antenna', 'vel-n', 'vel-e', 'vel-u', 'hdop', 'vdop', 'tdop', 'rms',
    'sigma-n', 'sigma-e', 'sigma-u-detail', 'unit-std', 'clock-offset', 'clock-drift',
  ];
  if (!pos) {
    clearIds.forEach(id => set(id, '—'));
  } else {
    set('lat', fmtDMS(pos.latitude, true));
    set('lon', fmtDMS(pos.longitude, false));
    set('alt', fmtM(pos.altitude, 2, 'm'));
    set('sigma-h', fmtSigma(pos.sigmaH ?? horizontalSigma(pos.sigmaN, pos.sigmaE)));
    set('sigma-u', fmtSigma(pos.sigmaU));
    set('data-age', fmtAge(pos));
    set('svs', `${pos.svsUsed} / ${pos.svsTracked}`);
    set('aug', pos.augmentationText || String(pos.augmentation));

    set('vel-n', fmtVel(pos.velocityN));
    set('vel-e', fmtVel(pos.velocityE));
    set('vel-u', fmtVel(pos.velocityU));
    set('hdop', fmtM(pos.hdop, 2, ''));
    set('vdop', fmtM(pos.vdop, 2, ''));
    set('tdop', fmtM(pos.tdop, 2, ''));
    set('rms', fmtSigma(pos.rms));
    set('sigma-n', fmtSigma(pos.sigmaN));
    set('sigma-e', fmtSigma(pos.sigmaE));
    set('sigma-u-detail', fmtSigma(pos.sigmaU));
    set('unit-std', fmtSigma(pos.unitStdDev));
    set('clock-offset', fmtM(pos.clockOffset, 6, 's'));
    set('clock-drift', fmtM(pos.clockDrift, 3, 's/s'));
  }
  if (!dualAntenna) {
    set('antenna', rt27?.antennas || '—');
  }
}

function horizontalSigma(sigmaN, sigmaE) {
  const n = sigmaN ?? 0;
  const e = sigmaE ?? 0;
  return Math.hypot(n, e);
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
      return a.svid - b.svid || (a.antenna ?? 0) - (b.antenna ?? 0) || a.system - b.system;
    case 'antenna':
      return (a.antenna ?? 0) - (b.antenna ?? 0) || a.system - b.system || a.svid - b.svid;
    case 'used': {
      const au = a.usedInSolution ? 1 : 0;
      const bu = b.usedInSolution ? 1 : 0;
      return au - bu || a.svid - b.svid || a.system - b.system;
    }
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

function fmtBandCell(band, col, row) {
  const slipLevel = displaySlipLevel(band?._slipLevel, row);
  const slipCls = slipCellClass(slipLevel);
  const missingL1 = col === 'l1' && !bandHasData(band) && rowHasNonL1Band(row);
  const cls = `band col-${col}${slipCls}${missingL1 ? ' l1-missing' : ''}`;
  const title = missingL1
    ? 'No L1 tracking — other bands present'
    : bandHoverText(band, col, row);
  if (!band?.present || !band.signals?.length) {
    return fmtTd(col, `${cls} empty`, '—', title);
  }
  const snrs = band.signals.map(s => s.snr.toFixed(1)).join('/');
  const tracks = band.signals.map(s => fmtTrackLabel(s)).join('/');
  let slipLine = '';
  if (showSlipCounts && slipIndicated(row)) {
    const counts = band.signals.map(s => String(s._slipCount ?? 0)).join('/');
    slipLine = `<span class="slip-val">${counts}</span>`;
  }
  const body = `<span class="snr-val">${snrs}</span><span class="track-val">${tracks}</span>${slipLine}`;
  return fmtTd(col, cls, body, title);
}

function fmtAntennaCell(row) {
  const ant = row.antenna ?? 0;
  return fmtTd('antenna', 'col-antenna', String(ant), `Antenna ${ant}`);
}

function updateSNRTable(rows) {
  snrBody.innerHTML = '';
  updateSortHeaders();
  updateEmptyColumns(rows);
  updateAntennaColumnVisibility();
  if (!rows.length) return;

  for (const row of sortedRows(rows)) {
    const tr = document.createElement('tr');
    if (row.raimFault) tr.classList.add('sv-raim');
    tr.innerHTML = `
      ${fmtTd('system', 'col-system', escapeHtml(row.systemName), `${row.systemName} · system ${row.system}`)}
      ${fmtSVCell(row)}
      ${fmtAntennaCell(row)}
      ${fmtUsedCell(row)}
      ${fmtTd('elevation', 'col-elevation', String(row.elevation), `Elevation ${row.elevation}°`)}
      ${fmtTd('azimuth', 'col-azimuth', String(positiveAz(row.azimuth)), `Azimuth ${positiveAz(row.azimuth)}°`)}
      ${fmtBandCell(row.l1, 'l1', row)}
      ${fmtBandCell(row.l2, 'l2', row)}
      ${fmtBandCell(row.l5, 'l5', row)}
      ${fmtBandCell(row.l6, 'l6', row)}`;
    snrBody.appendChild(tr);
  }
  updateEmptyColumns(rows);
}

function refreshDataView() {
  const rows = visibleRows();
  drawSky(rows);
  updateSNRTable(rows);
  updateLegendUI();
  updateDualAntennaUI();
}

function escapeHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;');
}

function applySnapshot(snap) {
  statusEl.textContent = snap.connected ? 'Connected' : 'Disconnected';
  statusEl.className = 'status ' + (snap.connected ? 'connected' : 'disconnected');

  countsEl.textContent = `packets: ${snap.packetCount} · raw: ${snap.rawCount}`;
  portEl.textContent = formatPortLine(snap);
  errorEl.textContent = snap.lastError || '';

  hasPositionSVData = (snap.position?.svs?.length ?? 0) > 0;

  if (snap.rt27) {
    allRows = snap.rt27.svs || [];
    dualAntenna = (snap.rt27.antennaCount ?? 0) >= 2;
    if (!dualAntenna) filterAntennaMode = 'both';
    enrichSlipState(snap.rt27, allRows);
    const t = snap.rt27.timeSec.toFixed(3);
    const total = allRows.length;
    const shown = visibleRows().length;
    let epoch = `GPS week ${snap.rt27.week} · t=${t}s · ${snap.rt27.numSVs} SVs`;
    if (hasActiveFilters()) {
      if (shown !== total) epoch += ` · showing ${shown}`;
    }
    epochEl.textContent = epoch;
    refreshDataView();
  } else if (snap.position && allRows.length) {
    mergePositionFlags(snap.position, allRows);
    refreshDataView();
  }

  updatePosition(snap.position, snap.rt27);
}

function mergePositionFlags(pos, rows) {
  if (!pos?.svs?.length) return;
  const lookup = new Map();
  for (const sv of pos.svs) {
    lookup.set(`${sv.system}:${sv.svid}`, sv);
  }
  for (const row of rows) {
    const sv = lookup.get(`${row.system}:${row.svid}`);
    row.usedInSolution = sv?.usedInSolution ?? false;
    row.raimFault = sv?.raimFault ?? false;
    row.unhealthy = sv?.unhealthy ?? false;
  }
}

document.querySelectorAll('#snr-table th.sortable').forEach(th => {
  th.dataset.label = th.textContent.trim();
  th.addEventListener('click', () => {
    const col = th.dataset.sort;
    if (sortCol === col) sortDir = -sortDir;
    else {
      sortCol = col;
      sortDir = (col === 'l1' || col === 'l2' || col === 'l5' || col === 'l6' || col === 'used') ? -1 : 1;
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

if (themeSelectEl) {
  if (!['system', 'light', 'dark'].includes(themeMode)) themeMode = 'system';
  themeSelectEl.value = themeMode;
  applyTheme(themeMode);
  themeSelectEl.addEventListener('change', () => applyTheme(themeSelectEl.value));
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (themeMode === 'system') refreshDataView();
  });
} else {
  syncThemeColors();
}

loadServerConfig();
connectSSE();
initSkyLegend();
updateSortHeaders();

function formatPortLine(snap) {
  if (snap.port) return `port: ${snap.port}`;
  if (serverConfig.demo) return 'demo mode';
  if (serverConfig.hosted) return snap.connected ? 'connected' : 'not connected';
  return '—';
}

async function refreshServerConfig() {
  try {
    serverConfig = await fetch('/api/config').then(r => r.json());
    applyShowDev(!!serverConfig.showDev);
  } catch (_) { /* ignore */ }
}

function loadServerConfig() {
  fetch('/api/config')
    .then(r => r.json())
    .then(cfg => {
      serverConfig = cfg;
      applyShowDev(!!cfg.showDev);
      if (cfg.hosted) {
        connectFormEl.hidden = false;
        restoreConnectionFields(cfg);
        if (!cfg.allowLocalHosts) {
          connectHostEl.title = 'Local and private addresses are blocked unless the server is started with -allow-local-hosts';
        }
      }
    })
    .catch(() => {});
}

function showConnectError(msg) {
  if (!msg) {
    connectErrorEl.hidden = true;
    connectErrorEl.textContent = '';
    return;
  }
  connectErrorEl.hidden = false;
  connectErrorEl.textContent = msg;
}

async function postJSON(url, body) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body ?? {}),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

connectHostEl?.addEventListener('input', () => persistConnectionDraft());
connectPortEl?.addEventListener('input', () => persistConnectionDraft());

connectFormEl?.addEventListener('submit', async (ev) => {
  ev.preventDefault();
  showConnectError('');
  const host = connectHostEl.value.trim();
  const port = parseInt(connectPortEl.value, 10);
  persistConnectionDraft(host, port);
  try {
    await postJSON('/api/connect', { host, port });
    await refreshServerConfig();
  } catch (err) {
    showConnectError(err.message);
  }
});

connectDisconnectEl?.addEventListener('click', async () => {
  showConnectError('');
  try {
    await postJSON('/api/disconnect');
    await refreshServerConfig();
  } catch (err) {
    showConnectError(err.message);
  }
});

if (showSlipCountsEl) {
  showSlipCountsEl.checked = showSlipCounts;
  showSlipCountsEl.addEventListener('change', () => {
    showSlipCounts = showSlipCountsEl.checked;
    localStorage.setItem('showSlipCounts', showSlipCounts ? '1' : '0');
    updateSNRTable(visibleRows());
  });
}

if (showTrackTypesEl) {
  showTrackTypesEl.checked = showTrackTypes;
  showTrackTypesEl.addEventListener('change', () => {
    showTrackTypes = showTrackTypesEl.checked;
    localStorage.setItem(SHOW_TRACK_TYPES_KEY, showTrackTypes ? '1' : '0');
    updateSNRTable(visibleRows());
  });
}

if (slipMaskEl) {
  slipMaskEl.value = String(slipMask);
  const applySlipMask = () => {
    slipMask = clampSlipMask(slipMaskEl.value);
    slipMaskEl.value = String(slipMask);
    localStorage.setItem(SLIP_MASK_KEY, String(slipMask));
    refreshDataView();
  };
  slipMaskEl.addEventListener('change', applySlipMask);
  slipMaskEl.addEventListener('input', applySlipMask);
}

if (antennaFilterEl) {
  if (!['both', '0', '1'].includes(filterAntennaMode)) filterAntennaMode = 'both';
  antennaFilterEl.querySelectorAll('.antenna-filter-btn').forEach(btn => {
    btn.addEventListener('click', () => setAntennaFilter(btn.dataset.ant));
  });
}

filterUsedEl?.addEventListener('click', toggleUsedFilter);

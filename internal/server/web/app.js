const sky = document.getElementById('sky');
const ctx = sky.getContext('2d');
const statusEl = document.getElementById('status');
const epochEl = document.getElementById('epoch');
const snrBody = document.getElementById('snr-body');
const countsEl = document.getElementById('counts');
const portEl = document.getElementById('port');
const errorEl = document.getElementById('error');

const systemColors = {
  GPS: '#58a6ff',
  GLONASS: '#f85149',
  Galileo: '#a371f7',
  Beidou: '#ffa657',
  QZSS: '#79c0ff',
  SBAS: '#3fb950',
  IRNSS: '#d2a8ff',
  '?': '#8b949e',
};

function rad(deg) { return (deg * Math.PI) / 180; }

function drawSky(signals) {
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

  if (!signals || !signals.length) return;

  for (const s of signals) {
    const el = s.elevation ?? 0;
    const az = s.azimuth ?? 0;
    const rr = r * (1 - el / 90);
    const a = rad(az - 90);
    const x = cx + rr * Math.cos(a);
    const y = cy + rr * Math.sin(a);
    const color = systemColors[s.systemName] || systemColors['?'];

    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(x, y, 7, 0, Math.PI * 2);
    ctx.fill();

    ctx.fillStyle = '#e6edf3';
    ctx.font = '10px system-ui';
    ctx.textAlign = 'center';
    ctx.fillText(String(s.svid), x, y + 3);
  }
}

function fmtDeg(radVal) {
  return (radVal * 180 / Math.PI).toFixed(6) + '°';
}

function updatePosition(pos) {
  const set = (id, val) => { document.getElementById(id).textContent = val; };
  if (!pos) {
    ['lat', 'lon', 'alt', 'hdop', 'rms', 'svs', 'aug'].forEach(id => set(id, '—'));
    return;
  }
  set('lat', fmtDeg(pos.latitude));
  set('lon', fmtDeg(pos.longitude));
  set('alt', pos.altitude.toFixed(2) + ' m');
  set('hdop', pos.hdop.toFixed(2));
  set('rms', pos.rms.toFixed(4) + ' m');
  set('svs', `${pos.svsUsed} / ${pos.svsTracked}`);
  set('aug', String(pos.augmentation));
}

function updateSNRTable(signals) {
  snrBody.innerHTML = '';
  if (!signals) return;
  const sorted = [...signals].sort((a, b) => b.snr - a.snr);
  for (const s of sorted) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${s.systemName}</td>
      <td>${s.svid}</td>
      <td>${s.elevation}</td>
      <td>${s.azimuth}</td>
      <td>${s.block}</td>
      <td>${s.snr.toFixed(1)}</td>`;
    snrBody.appendChild(tr);
  }
}

function applySnapshot(snap) {
  statusEl.textContent = snap.connected ? 'Connected' : 'Disconnected';
  statusEl.className = 'status ' + (snap.connected ? 'connected' : 'disconnected');

  countsEl.textContent = `packets: ${snap.packetCount} · raw: ${snap.rawCount}`;
  portEl.textContent = snap.port ? `port: ${snap.port}` : 'demo mode';
  errorEl.textContent = snap.lastError || '';

  if (snap.rt27) {
    const t = snap.rt27.timeSec.toFixed(3);
    epochEl.textContent = `GPS week ${snap.rt27.week} · t=${t}s · ${snap.rt27.numSVs} SVs`;
    drawSky(snap.rt27.signals);
    updateSNRTable(snap.rt27.signals);
  }

  if (snap.position) {
    updatePosition(snap.position);
  }
}

function connectSSE() {
  const es = new EventSource('/api/events');
  es.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === 'snapshot' && msg.snapshot) {
        applySnapshot(msg.snapshot);
      }
    } catch (_) { /* ignore */ }
  };
  es.onerror = () => {
    es.close();
    setTimeout(connectSSE, 2000);
  };
}

fetch('/api/snapshot')
  .then(r => r.json())
  .then(applySnapshot)
  .catch(() => {});

connectSSE();
drawSky([]);

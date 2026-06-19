package handlers

import (
	"net/http"
)

// HandleUI serves the log viewer web interface.
func (h *Handler) HandleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(logViewerHTML))
}

const logViewerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Download Log Viewer</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root {
    --bg:      #0d1117; --surface: #161b22; --border: #30363d;
    --text:    #e6edf3; --muted:   #8b949e; --accent:  #58a6ff;
    --green:   #3fb950; --yellow:  #d29922; --red:     #f85149;
    --orange:  #f0883e; --purple:  #bc8cff;
  }
  html, body { height: 100%; background: var(--bg); color: var(--text); font-family: 'Inter', sans-serif; font-size: 14px; }

  /* ── Layout ── */
  .layout { display: flex; width: 100%; height: 100vh; overflow: hidden; }
  .sidebar {
    width: 240px; flex-shrink: 0; background: var(--surface);
    border-right: 1px solid var(--border); display: flex; flex-direction: column; overflow-y: auto;
  }
  .sidebar-section { padding: 16px 0 4px; }
  .sidebar-title {
    font-size: 11px; font-weight: 600; letter-spacing: .08em;
    color: var(--muted); text-transform: uppercase; padding: 0 16px 8px;
    display: flex; align-items: center; gap: 6px;
  }
  .sidebar-title .count {
    background: var(--border); color: var(--muted); font-size: 10px;
    padding: 1px 6px; border-radius: 10px; font-weight: 500;
  }
  .file-item {
    padding: 7px 16px; cursor: pointer; font-family: 'JetBrains Mono', monospace; font-size: 12px;
    color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    border-left: 2px solid transparent; transition: all .15s;
    display: flex; align-items: center; gap: 8px;
  }
  .file-item:hover { color: var(--text); background: rgba(88,166,255,.06); }
  .file-item.active { color: var(--accent); border-left-color: var(--accent); background: rgba(88,166,255,.08); }
  .file-item .icon { flex-shrink: 0; }
  .file-item .name { flex: 1; overflow: hidden; text-overflow: ellipsis; }
  .file-item .size { font-size: 10px; color: var(--border); flex-shrink: 0; }

  .main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
  .header {
    padding: 10px 20px; border-bottom: 1px solid var(--border); background: var(--surface);
    display: flex; align-items: center; gap: 12px;
  }
  .header-title { font-weight: 600; font-size: 15px; flex: 1; }
  .header-file { font-family: 'JetBrains Mono', monospace; font-size: 12px; color: var(--muted); }
  .badge { padding: 2px 8px; border-radius: 20px; font-size: 11px; font-weight: 500; display: flex; align-items: center; gap: 5px; }
  .badge.live    { background: rgba(63,185,80,.15);  color: var(--green);  border: 1px solid rgba(63,185,80,.3); }
  .badge.offline { background: rgba(248,81,73,.15);  color: var(--red);    border: 1px solid rgba(248,81,73,.3); }
  .badge.waiting { background: rgba(210,153,34,.15); color: var(--yellow); border: 1px solid rgba(210,153,34,.3); }
  .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; animation: pulse 1.5s infinite; }
  @keyframes pulse { 0%,100%{opacity:1}50%{opacity:.3} }
  .badge.offline .dot, .badge.waiting .dot { animation: none; }
  .btn { padding: 5px 12px; border-radius: 6px; border: 1px solid var(--border); background: transparent; color: var(--text); font-size: 12px; cursor: pointer; transition: all .15s; font-family: inherit; }
  .btn:hover { background: rgba(255,255,255,.06); border-color: var(--accent); color: var(--accent); }

  .log-wrap { flex: 1; overflow-y: auto; padding: 12px 0; }
  .log-line { display: flex; font-family: 'JetBrains Mono', monospace; font-size: 12.5px; line-height: 1.65; padding: 1px 20px; transition: background .1s; }
  .log-line:hover { background: rgba(255,255,255,.03); }
  .log-num { color: var(--border); min-width: 40px; user-select: none; text-align: right; padding-right: 16px; flex-shrink: 0; }
  .log-text { flex: 1; white-space: pre-wrap; word-break: break-all; }
  .log-text.lvl-ok    { color: #3fb950; }
  .log-text.lvl-warn  { color: #d29922; }
  .log-text.lvl-err   { color: #f85149; }
  .log-text.lvl-info  { color: var(--text); }
  .log-text.lvl-muted { color: var(--muted); }

  .footer { border-top: 1px solid var(--border); padding: 6px 20px; font-size: 11px; color: var(--muted); display: flex; gap: 16px; align-items: center; }

  .empty-state { color: var(--muted); text-align: center; padding: 80px 20px; font-size: 14px; }

  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
  ::-webkit-scrollbar-thumb:hover { background: var(--muted); }
</style>
</head>
<body>

<div class="layout">
  <div class="sidebar">
    <div class="sidebar-section">
      <div class="sidebar-title">📋 Main Logs <span class="count" id="main-count">0</span></div>
      <div id="main-files"></div>
    </div>
    <div class="sidebar-section" id="process-section" style="display:none">
      <div class="sidebar-title">⚙️ Process Logs <span class="count" id="process-count">0</span></div>
      <div id="process-files"></div>
    </div>
  </div>
  <div class="main">
    <div class="header">
      <div class="header-title">📥 Download Log Viewer</div>
      <div class="header-file" id="current-file">—</div>
      <div class="badge waiting" id="ws-badge"><span class="dot"></span><span id="ws-label">Connecting...</span></div>
      <button class="btn" onclick="reconnect()">↺ Reconnect</button>
    </div>
    <div class="log-wrap" id="log-wrap">
      <div style="padding:40px 20px;color:var(--muted);font-family:'JetBrains Mono',monospace;font-size:13px;">Select a log file from the sidebar...</div>
    </div>
    <div class="footer">
      <span id="line-count">—</span>
      <span id="last-update">—</span>
    </div>
  </div>
</div>

<script>
// ── WebSocket ────────────────────────────────────────────────────────────
let ws = null, currentRoom = null;
function connect() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(proto + '//' + location.host + '/ws');
  ws.onopen = () => { setStatus('live', 'Live'); if (currentRoom) subscribe(currentRoom); };
  ws.onclose = () => { setStatus('offline', 'Disconnected'); setTimeout(connect, 3000); };
  ws.onerror = () => setStatus('offline', 'Error');
  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    if (msg.type === 'files') renderFileList(msg.files || []);
    if (msg.type === 'log' && msg.room === currentRoom) renderLog(msg.lines || [], msg.total || 0);
  };
}
function reconnect() { if (ws) ws.close(); }
function subscribe(filename) {
  currentRoom = filename;
  document.getElementById('current-file').textContent = filename;
  // Highlight active item
  document.querySelectorAll('.file-item').forEach(el => {
    el.classList.toggle('active', el.dataset.room === filename);
  });
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'subscribe', room: filename }));
}
function setStatus(cls, label) {
  document.getElementById('ws-badge').className = 'badge ' + cls;
  document.getElementById('ws-label').textContent = label;
}

function renderFileList(files) {
  const mainList = document.getElementById('main-files');
  const processList = document.getElementById('process-files');
  const processSection = document.getElementById('process-section');
  const prev = currentRoom;

  const mainFiles = files.filter(f => !f.name.startsWith('process/'));
  const processFiles = files.filter(f => f.name.startsWith('process/'));

  document.getElementById('main-count').textContent = mainFiles.length;
  document.getElementById('process-count').textContent = processFiles.length;
  processSection.style.display = processFiles.length ? '' : 'none';

  mainList.innerHTML = '';
  mainFiles.forEach((f, i) => {
    mainList.appendChild(createFileItem(f.name, f.name, f.size));
    if (i === 0 && !prev) subscribe(f.name);
  });

  processList.innerHTML = '';
  processFiles.forEach(f => {
    const displayName = f.name.replace('process/', '');
    processList.appendChild(createFileItem(f.name, displayName, f.size));
  });
}

function createFileItem(roomName, displayName, size) {
  const el = document.createElement('div');
  el.className = 'file-item' + (currentRoom === roomName ? ' active' : '');
  el.dataset.room = roomName;
  el.innerHTML = '<span class="icon">' + (roomName.startsWith('process/') ? '⚙️' : '📄') + '</span>'
    + '<span class="name">' + escHtml(displayName) + '</span>'
    + '<span class="size">' + fmtSize(size) + '</span>';
  el.onclick = () => subscribe(roomName);
  return el;
}

function renderLog(lines, total) {
  if (!lines.length) return;
  const wrap = document.getElementById('log-wrap');
  const atTop = wrap.scrollTop < 20;
  wrap.innerHTML = lines.map((line, i) =>
    '<div class="log-line"><span class="log-num">' + (total-i) + '</span><span class="log-text ' + classify(line) + '">' + escHtml(line) + '</span></div>'
  ).join('');
  if (atTop) wrap.scrollTop = 0;
  document.getElementById('line-count').textContent = total + ' total, showing ' + lines.length;
  document.getElementById('last-update').textContent = 'Updated ' + new Date().toLocaleTimeString();
}

function classify(line) {
  if (/✅|→ active|connected|ensured|successful|COMPLETE/i.test(line)) return 'lvl-ok';
  if (/⚠️|warn|failed|skip/i.test(line)) return 'lvl-warn';
  if (/❌|error|fatal/i.test(line)) return 'lvl-err';
  if (/⏳|pending|mismatch|waiting/i.test(line)) return 'lvl-muted';
  return 'lvl-info';
}
function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function fmtSize(b) {
  if (!b) return '0 B';
  if (b < 1024) return b + ' B';
  if (b < 1024*1024) return (b/1024).toFixed(1) + ' KB';
  return (b/1024/1024).toFixed(1) + ' MB';
}

connect();
</script>
</body>
</html>`

import { useState, useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';
import { Monitor, Terminal, Trash2, Wifi, GripVertical, Pencil, Check, X } from 'lucide-react';
import Sparkline from '../charts/Sparkline';
import { deleteNode, renameNode } from '../../lib/api';

function getOSLabel(os) {
  if (!os) return 'Unknown OS';
  const l = os.toLowerCase();
  if (l.includes('windows 10')) return 'Windows 10';
  if (l.includes('windows server 2022')) return 'Win Server 2022';
  if (l.includes('windows server 2019')) return 'Win Server 2019';
  if (l.includes('windows server')) return 'Windows Server';
  if (l.includes('windows')) return 'Windows';
  if (l.includes('ubuntu')) return 'Ubuntu';
  if (l.includes('astra')) return 'Astra Linux';
  if (l.includes('red') || l.includes('рос')) return 'РЕД ОС';
  return os.split(' ').slice(0, 2).join(' ');
}

function colorByPct(pct) {
  if (pct > 85) return 'text-red-400';
  if (pct > 60) return 'text-amber-400';
  return 'text-emerald-400';
}

function barColorClass(pct) {
  if (pct > 85) return 'bg-red-500';
  if (pct > 60) return 'bg-amber-500';
  return 'bg-blue-500';
}

function fmtBytes(bytes) {
  if (!bytes || bytes <= 0) return '0';
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)}M`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)}K`;
  return `${Math.round(bytes)}`;
}

function MiniBar({ value, label, right }) {
  const safe = Math.min(Math.max(value || 0, 0), 100);
  return (
    <div>
      <div className="flex justify-between text-xs mb-1">
        <span className="text-slate-500">{label}</span>
        <span className={`tabular-nums font-medium ${colorByPct(safe)}`}>{right}</span>
      </div>
      <div className="w-full bg-slate-700/60 h-1.5 rounded-full overflow-hidden">
        <div className={`${barColorClass(safe)} h-full rounded-full transition-all duration-500`}
          style={{ width: `${safe}%` }} />
      </div>
    </div>
  );
}

function ProbeDot({ label, active, ms }) {
  return (
    <div className={`flex items-center gap-1.5 px-2 py-1 rounded-lg border text-xs font-medium
      ${active
        ? 'bg-emerald-500/8 border-emerald-500/20 text-emerald-400'
        : 'bg-red-500/8 border-red-500/20 text-red-400/70'}`}>
      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${active ? 'bg-emerald-400' : 'bg-red-500/60'}`} />
      {label}
      {active && ms > 0 && <span className="text-emerald-500/60 text-[10px]">{ms < 1 ? '<1' : Math.round(ms)}ms</span>}
    </div>
  );
}

function AgentVersionBadge({ version, serverVersion }) {
  if (!version || version === 'unknown') return null;
  const upToDate = serverVersion && version === serverVersion;
  return (
    <span className={`text-[10px] px-1.5 py-0.5 rounded font-mono font-medium
      ${upToDate
        ? 'bg-emerald-500/10 text-emerald-400/80'
        : 'bg-amber-500/10 text-amber-400'}`}>
      {version}
    </span>
  );
}

export default function NodeCard({ node, onDeleted, dragHandleProps, isDragging, serverVersion }) {
  const isWindows = node.os?.toLowerCase().includes('windows');
  const ramPct = node.ramTotal > 0 ? (node.ramUsed / node.ramTotal) * 100 : 0;
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState('');
  const renameInputRef = useRef(null);
  const wasDraggingRef = useRef(false);

  useEffect(() => {
    if (isDragging) wasDraggingRef.current = true;
  }, [isDragging]);

  async function handleDelete(e) {
    e.preventDefault();
    e.stopPropagation();
    if (!confirming) { setConfirming(true); return; }
    setDeleting(true);
    try {
      await deleteNode(node.name);
      onDeleted?.();
    } catch {
      setDeleting(false);
      setConfirming(false);
    }
  }

  function handleCancelConfirm(e) {
    e.preventDefault();
    e.stopPropagation();
    setConfirming(false);
  }

  function handleStartRename(e) {
    e.preventDefault();
    e.stopPropagation();
    setRenameValue(node.displayName || node.name);
    setRenaming(true);
    setTimeout(() => renameInputRef.current?.select(), 0);
  }

  async function handleConfirmRename(e) {
    e.preventDefault();
    e.stopPropagation();
    const alias = renameValue.trim();
    try {
      await renameNode(node.name, alias === node.name ? '' : alias);
    } catch { /* ignore */ }
    setRenaming(false);
  }

  function handleCancelRename(e) {
    e.preventDefault();
    e.stopPropagation();
    setRenaming(false);
  }

  const netTotal = (node.netRecvSec || 0) + (node.netSentSec || 0);

  return (
    <Link
      to={`/node/${node.name}`}
      onClick={e => {
        if (wasDraggingRef.current) {
          wasDraggingRef.current = false;
          e.preventDefault();
        }
      }}
      className="group bg-slate-800/80 backdrop-blur-sm rounded-2xl border border-slate-700/50
                 hover:border-blue-500/40 hover:bg-slate-800 hover:shadow-xl hover:shadow-blue-500/5
                 transition-all duration-200 block relative overflow-hidden"
    >
      {/* Цветная полоска сверху по статусу */}
      <div className={`h-0.5 w-full ${node.online ? 'bg-gradient-to-r from-emerald-500/60 to-blue-500/40' : 'bg-slate-700'}`} />

      <div className="p-5">
        {/* ── Заголовок ── */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-2.5">
            {dragHandleProps && (
              <div
                {...dragHandleProps}
                onClick={e => e.preventDefault()}
                className="flex-shrink-0 p-1 -ml-1 rounded text-slate-600 hover:text-slate-400 cursor-grab active:cursor-grabbing opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <GripVertical className="w-4 h-4" />
              </div>
            )}
            <div className="relative flex-shrink-0">
              <div className={`p-2 rounded-xl ${isWindows ? 'bg-blue-500/10' : 'bg-emerald-500/10'}`}>
                {isWindows
                  ? <Monitor className="w-4 h-4 text-blue-400" />
                  : <Terminal className="w-4 h-4 text-emerald-400" />}
              </div>
              <span className={`absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-slate-800
                ${node.online ? 'bg-emerald-400' : 'bg-red-400'}`} />
            </div>
            <div className="min-w-0">
              {renaming ? (
                <div className="flex items-center gap-1" onClick={e => e.preventDefault()}>
                  <input
                    ref={renameInputRef}
                    value={renameValue}
                    onChange={e => setRenameValue(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') handleConfirmRename(e); if (e.key === 'Escape') handleCancelRename(e); }}
                    className="text-sm font-semibold bg-slate-700 text-slate-100 rounded px-1.5 py-0.5 outline-none border border-blue-500/50 w-32"
                    maxLength={64}
                  />
                  <button onClick={handleConfirmRename} className="text-emerald-400 hover:text-emerald-300"><Check className="w-3.5 h-3.5" /></button>
                  <button onClick={handleCancelRename} className="text-slate-500 hover:text-slate-300"><X className="w-3.5 h-3.5" /></button>
                </div>
              ) : (
                <div className="flex items-center gap-1 group/name">
                  <div className="font-semibold text-slate-100 text-sm leading-tight truncate max-w-[130px]">
                    {node.displayName || node.name}
                  </div>
                  <button
                    onClick={handleStartRename}
                    className="opacity-0 group-hover/name:opacity-100 transition-opacity text-slate-600 hover:text-slate-400"
                  >
                    <Pencil className="w-3 h-3" />
                  </button>
                </div>
              )}
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className="text-xs text-slate-500">{getOSLabel(node.os)}</span>
                <AgentVersionBadge version={node.agentVersion} serverVersion={serverVersion} />
              </div>
            </div>
          </div>

          <div className="flex items-center gap-1.5 flex-shrink-0">
            <span className={`text-xs px-2 py-0.5 rounded-full font-medium
              ${node.online ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
              {node.online ? 'Online' : 'Offline'}
            </span>

            {!node.online && (
              confirming ? (
                <div className="flex items-center gap-1" onClick={e => e.preventDefault()}>
                  <button onClick={handleDelete} disabled={deleting}
                    className="text-xs px-2 py-0.5 rounded-md bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30 transition-all font-medium">
                    {deleting ? '...' : 'Да'}
                  </button>
                  <button onClick={handleCancelConfirm}
                    className="text-xs px-2 py-0.5 rounded-md bg-slate-700/50 text-slate-400 hover:bg-slate-700 border border-slate-600/30 transition-all">
                    Нет
                  </button>
                </div>
              ) : (
                <button onClick={handleDelete} title="Удалить узел"
                  className="opacity-0 group-hover:opacity-100 p-1 rounded-md text-slate-600 hover:text-red-400 hover:bg-red-500/10 transition-all duration-200">
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              )
            )}
          </div>
        </div>

        {/* IP + Uptime */}
        <div className="flex items-center gap-2 text-xs text-slate-500 mb-4">
          <span className="font-mono">{node.ip || '—'}</span>
          {node.online ? (
            <><span className="text-slate-700">·</span><span>⬆ {node.uptime || '0 ч.'}</span></>
          ) : node.lastSeen ? (
            <><span className="text-slate-700">·</span><span className="text-red-400/60">был {node.lastSeen}</span></>
          ) : null}
        </div>

        {/* ── Метрики ── */}
        <div className={`space-y-2.5 ${!node.online ? 'opacity-35 pointer-events-none' : ''}`}>
          <MiniBar label="CPU" value={node.cpu}
            right={`${node.cpu || 0}%`} />
          <MiniBar label="RAM" value={ramPct}
            right={`${(node.ramUsed || 0).toFixed(1)} / ${(node.ramTotal || 0).toFixed(1)} GB`} />
          <MiniBar label="Disk" value={node.diskUsage}
            right={`${(node.diskUsage || 0).toFixed(1)}%`} />

          {/* Сеть + TCP */}
          <div className="flex items-center justify-between pt-0.5">
            <div className="flex items-center gap-1 text-xs text-slate-500">
              <Wifi className="w-3 h-3" />
              <span className="text-cyan-400 tabular-nums">↓{fmtBytes(node.netRecvSec)}</span>
              <span className="text-blue-400 tabular-nums">↑{fmtBytes(node.netSentSec)}</span>
              <span className="text-slate-600 text-[10px]">B/s</span>
            </div>
            {(node.tcpTotal || 0) > 0 && (
              <span className="text-xs text-slate-600">
                TCP: <span className="text-slate-400">{node.tcpEstablished || 0}</span>
                <span className="text-slate-700">/{node.tcpTotal}</span>
              </span>
            )}
          </div>
        </div>

        {/* ── Sparkline ── */}
        <div className="mt-3 h-10">
          <Sparkline data={node.cpuHistory} color={node.online ? '#3b82f6' : '#334155'} />
        </div>

        {/* ── Сервисные пробы ── */}
        <div className="mt-3 flex flex-wrap gap-1.5">
          <ProbeDot label="SSH"  active={node.sshReachable}  ms={node.sshMs} />
          {isWindows && <ProbeDot label="RDP"   active={node.rdpReachable}   ms={node.rdpMs} />}
          {isWindows && <ProbeDot label="SMB"   active={node.smbReachable}   ms={node.smbMs} />}
          <ProbeDot label="HTTP" active={node.httpReachable} ms={node.httpMs} />
          {isWindows && <ProbeDot label="WinRM" active={node.winrmReachable} ms={node.winrmMs} />}
          <ProbeDot label="DNS"  active={node.dnsReachable}  ms={node.dnsMs} />
          {(node.cpuTemp || 0) > 0 && (
            <span className="text-xs text-amber-400/80 ml-auto self-center">
              🌡 {Math.round(node.cpuTemp)}°C
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}

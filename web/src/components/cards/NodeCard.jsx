import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Monitor, Terminal, Trash2 } from 'lucide-react';
import Sparkline from '../charts/Sparkline';
import { deleteNode } from '../../lib/api';

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

function MiniBar({ value }) {
  const safe = Math.min(Math.max(value || 0, 0), 100);
  return (
    <div className="w-full bg-slate-700/60 h-1.5 rounded-full overflow-hidden">
      <div className={`${barColorClass(safe)} h-full rounded-full transition-all duration-500`} style={{ width: `${safe}%` }} />
    </div>
  );
}

function ServiceDot({ label, active }) {
  return (
    <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium
      ${active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${active ? 'bg-emerald-400' : 'bg-red-400'}`} />
      {label}
    </span>
  );
}

export default function NodeCard({ node, onDeleted }) {
  const isWindows = node.os?.toLowerCase().includes('windows');
  const ramPct = node.ramTotal > 0 ? (node.ramUsed / node.ramTotal) * 100 : 0;
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

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

  return (
    <Link
      to={`/node/${node.name}`}
      className="group bg-slate-800/80 backdrop-blur-sm rounded-2xl p-5 border border-slate-700/50
                 hover:border-blue-500/40 hover:bg-slate-800 hover:shadow-xl hover:shadow-blue-500/5
                 transition-all duration-200 block relative"
    >
      {/* Заголовок */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2.5">
          <div className="relative">
            <div className={`p-2 rounded-lg ${isWindows ? 'bg-blue-500/10' : 'bg-emerald-500/10'}`}>
              {isWindows
                ? <Monitor className="w-4 h-4 text-blue-400" />
                : <Terminal className="w-4 h-4 text-emerald-400" />}
            </div>
            <span className={`absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-slate-800
              ${node.online ? 'bg-emerald-400' : 'bg-red-400'}`} />
          </div>
          <div>
            <div className="font-semibold text-slate-100 text-sm leading-none mb-0.5">{node.name}</div>
            <div className="text-xs text-slate-500">{getOSLabel(node.os)}</div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium
            ${node.online ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
            {node.online ? 'Online' : 'Offline'}
          </span>

          {/* Кнопка удаления — только для offline */}
          {!node.online && (
            confirming ? (
              <div className="flex items-center gap-1" onClick={e => e.preventDefault()}>
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  className="text-xs px-2 py-0.5 rounded-md bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30 transition-all font-medium"
                >
                  {deleting ? '...' : 'Удалить'}
                </button>
                <button
                  onClick={handleCancelConfirm}
                  className="text-xs px-2 py-0.5 rounded-md bg-slate-700/50 text-slate-400 hover:bg-slate-700 border border-slate-600/30 transition-all"
                >
                  Отмена
                </button>
              </div>
            ) : (
              <button
                onClick={handleDelete}
                title="Удалить узел"
                className="opacity-0 group-hover:opacity-100 p-1 rounded-md text-slate-600 hover:text-red-400 hover:bg-red-500/10 transition-all duration-200"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            )
          )}
        </div>
      </div>

      {/* IP + Uptime / Last seen */}
      <div className="flex items-center gap-3 text-xs text-slate-500 mb-4">
        <span>{node.ip || '—'}</span>
        {node.online ? (
          <><span className="text-slate-600">·</span><span>⬆ {node.uptime || '0 ч.'}</span></>
        ) : node.lastSeen ? (
          <><span className="text-slate-600">·</span><span className="text-red-400/70">Был онлайн: {node.lastSeen}</span></>
        ) : null}
      </div>

      {/* Метрики */}
      <div className={`space-y-3 ${!node.online ? 'opacity-40 pointer-events-none' : ''}`}>
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-400">CPU</span>
            <span className={`font-medium tabular-nums ${colorByPct(node.cpu)}`}>{node.cpu}%</span>
          </div>
          <MiniBar value={node.cpu} />
        </div>
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-400">RAM</span>
            <span className="text-slate-300 tabular-nums">
              <span className={colorByPct(ramPct)}>{node.ramUsed?.toFixed(1)}</span>
              <span className="text-slate-500"> / {node.ramTotal?.toFixed(1)} GB</span>
            </span>
          </div>
          <MiniBar value={ramPct} />
        </div>
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-400">Disk</span>
            <span className={`font-medium tabular-nums ${colorByPct(node.diskUsage)}`}>
              {node.diskUsage?.toFixed(1)}%
            </span>
          </div>
          <MiniBar value={node.diskUsage} />
        </div>
      </div>

      <div className="mt-4 h-10">
        <Sparkline data={node.cpuHistory} color={node.online ? '#3b82f6' : '#475569'} />
      </div>

      <div className="mt-3 flex flex-wrap gap-1.5">
        {isWindows ? (
          <>
            <ServiceDot label="RDP" active={node.rdpRunning} />
            <ServiceDot label="SMB" active={node.smbRunning} />
          </>
        ) : (
          <ServiceDot label="SSH" active={node.online} />
        )}
      </div>
    </Link>
  );
}

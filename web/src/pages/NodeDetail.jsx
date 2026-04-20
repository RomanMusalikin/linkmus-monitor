import { useParams, Link, useNavigate } from 'react-router-dom';
import { useState } from 'react';
import {
  ArrowLeft, Cpu, Database, HardDrive, Globe,
  Activity, Monitor, Terminal, List, Shield, TrendingUp,
  Wifi, MemoryStick, Trash2
} from 'lucide-react';
import { deleteNode } from '../lib/api';
import { useNodes } from '../hooks/useNodes';
import CpuGauge from '../components/charts/CpuGauge';
import CpuHistory from '../components/charts/CpuHistory';
import NetworkLines from '../components/charts/NetworkLines';
import ProgressBar from '../components/common/ProgressBar';
import { AreaChart, Area, ResponsiveContainer, Tooltip, XAxis } from 'recharts';

// ─── Утилиты ───────────────────────────────────────────────────────────────

function fmtBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B/s';
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB/s`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB/s`;
  return `${Math.round(bytes)} B/s`;
}

function colorByPct(pct) {
  if (pct > 85) return 'text-red-400';
  if (pct > 60) return 'text-amber-400';
  return 'text-emerald-400';
}

function barColor(pct) {
  if (pct > 85) return 'bg-red-500';
  if (pct > 60) return 'bg-amber-500';
  return 'bg-blue-500';
}

function getOSLabel(os) {
  if (!os) return 'Unknown OS';
  const l = os.toLowerCase();
  if (l.includes('windows server 2022')) return 'Windows Server 2022';
  if (l.includes('windows server 2019')) return 'Windows Server 2019';
  if (l.includes('windows server')) return 'Windows Server';
  if (l.includes('windows 10')) return 'Windows 10';
  if (l.includes('windows')) return 'Windows';
  if (l.includes('ubuntu')) return 'Ubuntu';
  if (l.includes('astra')) return 'Astra Linux';
  if (l.includes('red') || l.includes('рос')) return 'РЕД ОС';
  return os;
}

// ─── Мини-компоненты ───────────────────────────────────────────────────────

function Card({ title, icon: Icon, iconColor = 'text-blue-400', children, className = '' }) {
  return (
    <div className={`bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5 h-full ${className}`}>
      <div className="flex items-center gap-2 mb-4">
        <Icon className={`w-4 h-4 ${iconColor}`} />
        <span className="text-sm font-semibold text-slate-300">{title}</span>
      </div>
      {children}
    </div>
  );
}

function TopStat({ label, value, sub, color = 'text-slate-100' }) {
  return (
    <div className="bg-slate-900/50 border border-slate-700/40 rounded-xl p-3 text-center">
      <div className={`text-lg font-bold tabular-nums ${color}`}>{value}</div>
      {sub && <div className="text-xs text-slate-600 mt-0.5">{sub}</div>}
      <div className="text-xs text-slate-500 mt-0.5">{label}</div>
    </div>
  );
}

function InfoRow({ label, value }) {
  return (
    <div className="flex justify-between items-center py-1.5 border-b border-slate-700/30 last:border-0">
      <span className="text-xs text-slate-500">{label}</span>
      <span className="text-xs text-slate-300 font-medium tabular-nums text-right max-w-[180px] truncate">{value}</span>
    </div>
  );
}

function ServiceBadge({ label, port, active, ms }) {
  return (
    <div className={`flex items-center justify-between rounded-xl p-3 border
      ${active ? 'bg-emerald-500/5 border-emerald-500/20' : 'bg-red-500/5 border-red-500/20'}`}>
      <div className="flex items-center gap-2.5">
        <div className={`w-2 h-2 rounded-full ${active ? 'bg-emerald-400' : 'bg-red-400'}`} />
        <div>
          <div className="text-sm font-medium text-slate-200">{label}</div>
          {port && <div className="text-xs text-slate-500">:{port}</div>}
        </div>
      </div>
      <div className="flex items-center gap-2">
        {active && ms > 0 && (
          <span className="text-xs text-slate-500 tabular-nums">{Math.round(ms)} ms</span>
        )}
        <span className={`text-xs font-semibold px-2 py-0.5 rounded-md
          ${active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
          {active ? 'OK' : 'Down'}
        </span>
      </div>
    </div>
  );
}

// Карточка процессов с переключателем CPU / RAM
function ProcessesCard({ node, className }) {
  const [mode, setMode] = useState('cpu'); // 'cpu' | 'ram'

  const cpuProcs = node.processes || [];
  const ramProcs = node.topMemProcesses || [];

  // Объединяем все уникальные процессы по pid, берём данные из обоих списков
  const allPids = [...new Set([...cpuProcs.map(p => p.pid), ...ramProcs.map(p => p.pid)])];
  const cpuMap = Object.fromEntries(cpuProcs.map(p => [p.pid, p]));
  const ramMap = Object.fromEntries(ramProcs.map(p => [p.pid, p]));

  const merged = allPids.map(pid => {
    const c = cpuMap[pid] || {};
    const r = ramMap[pid] || {};
    return { pid, name: c.name || r.name, cpu: c.cpu ?? 0, ram: r.ram ?? c.ram ?? 0, user: c.user || r.user };
  });

  // Сортируем по активной метрике
  const rows = [...merged].sort((a, b) => mode === 'cpu' ? b.cpu - a.cpu : b.ram - a.ram).slice(0, 10);

  const maxCpu = Math.max(...rows.map(p => p.cpu), 0.1);
  const maxRam = Math.max(...rows.map(p => p.ram), 1);

  return (
    <Card title="Топ процессов" icon={List} iconColor="text-slate-400" className={className}>
      {/* Переключатель */}
      <div className="flex gap-1.5 mb-5">
        {[{ key: 'cpu', label: 'По CPU' }, { key: 'ram', label: 'По RAM' }].map(t => (
          <button
            key={t.key}
            onClick={() => setMode(t.key)}
            className={`px-4 py-1.5 rounded-lg text-xs font-semibold transition-all duration-200
              ${mode === t.key
                ? t.key === 'cpu'
                  ? 'bg-blue-500/15 text-blue-400 border border-blue-500/30 shadow shadow-blue-500/10'
                  : 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 shadow shadow-emerald-500/10'
                : 'text-slate-500 border border-transparent hover:text-slate-300 hover:border-slate-700'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {rows.length === 0
        ? <div className="text-slate-500 text-sm py-6 text-center">Данные о процессах ещё не поступали</div>
        : (
          <div className="space-y-1">
            {/* Заголовок */}
            <div className="grid grid-cols-[2rem_1fr_7rem_7rem] gap-2 px-2 pb-1 text-xs font-medium text-slate-600 uppercase tracking-wider">
              <span>#</span>
              <span>Процесс</span>
              <span className={`text-right transition-colors duration-300 ${mode === 'cpu' ? 'text-blue-500' : ''}`}>CPU %</span>
              <span className={`text-right transition-colors duration-300 ${mode === 'ram' ? 'text-emerald-500' : ''}`}>RAM</span>
            </div>

            {rows.map((proc, i) => {
              const cpuPct  = (proc.cpu / maxCpu) * 100;
              const ramPct  = (proc.ram / maxRam) * 100;
              const activePct = mode === 'cpu' ? cpuPct : ramPct;

              return (
                <div key={proc.pid}
                  className="group grid grid-cols-[2rem_1fr_7rem_7rem] gap-2 items-center
                             px-2 py-2 rounded-xl hover:bg-slate-700/25 transition-colors duration-150">

                  {/* Ранг */}
                  <span className="text-xs text-slate-600 tabular-nums font-medium">{i + 1}</span>

                  {/* Название + шкала активной метрики */}
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs font-semibold text-slate-200 truncate">{proc.name}</span>
                      <span className="text-xs text-slate-600 tabular-nums hidden sm:inline">{proc.pid}</span>
                    </div>
                    {/* Шкала — всегда под названием, цвет меняется при переключении */}
                    <div className="w-full bg-slate-700/50 h-1 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-500 ${mode === 'cpu' ? 'bg-blue-500' : 'bg-emerald-500'}`}
                        style={{ width: `${activePct}%` }}
                      />
                    </div>
                  </div>

                  {/* CPU */}
                  <div className="text-right">
                    <span className={`text-xs tabular-nums font-medium transition-all duration-300
                      ${mode === 'cpu' ? 'text-blue-400 text-sm' : 'text-slate-500'}`}>
                      {proc.cpu.toFixed(1)}%
                    </span>
                  </div>

                  {/* RAM */}
                  <div className="text-right">
                    <span className={`text-xs tabular-nums font-medium transition-all duration-300
                      ${mode === 'ram' ? 'text-emerald-400 text-sm' : 'text-slate-500'}`}>
                      {proc.ram >= 1024 ? `${(proc.ram / 1024).toFixed(1)} GB` : `${Math.round(proc.ram)} MB`}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )
      }
    </Card>
  );
}

// ─── Главный компонент ─────────────────────────────────────────────────────

export default function NodeDetail() {
  const { nodeId } = useParams();
  const navigate = useNavigate();
  const { data: nodes, loading, error } = useNodes();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function handleDelete() {
    if (!confirmDelete) { setConfirmDelete(true); return; }
    setDeleting(true);
    try {
      await deleteNode(nodeId);
      navigate('/');
    } catch {
      setDeleting(false);
      setConfirmDelete(false);
    }
  }

  if (loading && !nodes) {
    return (
      <div className="p-6 flex items-center justify-center h-full text-slate-400">
        <Activity className="w-5 h-5 animate-spin mr-2 text-blue-500" />
        Загрузка данных узла...
      </div>
    );
  }

  if (error) {
    return <div className="p-6 text-red-400">Ошибка: {error}</div>;
  }

  const node = nodes?.find(n => n.name === nodeId);
  if (!node) {
    return (
      <div className="p-6 text-slate-400">
        Узел <span className="text-slate-200 font-medium">{nodeId}</span> не найден.
      </div>
    );
  }

  const isWindows = node.os?.toLowerCase().includes('windows');
  const ramPct = node.ramTotal > 0 ? (node.ramUsed / node.ramTotal) * 100 : 0;
  const swapPct = node.swapTotal > 0 ? (node.swapUsed / node.swapTotal) * 100 : 0;
  const cpuCores = node.cpuCores || [];

  return (
    <div className="p-6 max-w-screen-2xl mx-auto">

      {/* ── Заголовок ── */}
      <div className="flex items-center gap-4 mb-6">
        <Link to="/"
          className="p-2 bg-slate-800 hover:bg-slate-700 rounded-xl border border-slate-700
                     text-slate-400 hover:text-slate-100 transition-colors flex-shrink-0">
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className={`p-2 rounded-xl ${isWindows ? 'bg-blue-500/10' : 'bg-emerald-500/10'}`}>
            {isWindows
              ? <Monitor className="w-5 h-5 text-blue-400" />
              : <Terminal className="w-5 h-5 text-emerald-400" />}
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-xl font-bold text-slate-100">{node.name}</h1>
              <span className={`w-2 h-2 rounded-full ${node.online ? 'bg-emerald-400' : 'bg-red-400'}`} />
              <span className={`text-xs px-2 py-0.5 rounded-full font-medium
                ${node.online ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
                {node.online ? 'Online' : `Offline · был ${node.lastSeen}`}
              </span>
            </div>
            <p className="text-slate-500 text-sm truncate">
              {getOSLabel(node.os)} · {node.ip || '—'} · ⬆ {node.uptime || '0 ч.'}
              {node.cpuModel ? ` · ${node.cpuModel}` : ''}
            </p>
          </div>
        </div>

        {/* Кнопка удаления — только для offline */}
        {!node.online && (
          <div className="flex items-center gap-2 ml-auto flex-shrink-0">
            {confirmDelete && (
              <button onClick={() => setConfirmDelete(false)}
                className="text-sm px-3 py-1.5 rounded-lg bg-slate-700 text-slate-400 hover:bg-slate-600 transition-all border border-slate-600">
                Отмена
              </button>
            )}
            <button
              onClick={handleDelete}
              disabled={deleting}
              className={`flex items-center gap-1.5 text-sm px-3 py-1.5 rounded-lg border transition-all font-medium
                ${confirmDelete
                  ? 'bg-red-500/20 text-red-400 border-red-500/40 hover:bg-red-500/30'
                  : 'bg-slate-800 text-slate-400 border-slate-700 hover:text-red-400 hover:border-red-500/30 hover:bg-red-500/10'}`}
            >
              <Trash2 className="w-4 h-4" />
              {deleting ? 'Удаление...' : confirmDelete ? 'Подтвердить удаление' : 'Удалить узел'}
            </button>
          </div>
        )}
      </div>

      {/* ── Быстрые показатели ── */}
      <div className="grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-8 gap-3 mb-6">
        <TopStat label="CPU" value={`${node.cpu}%`} color={colorByPct(node.cpu)} />
        <TopStat label="RAM" value={`${ramPct.toFixed(0)}%`}
          sub={`${(node.ramUsed || 0).toFixed(1)} / ${(node.ramTotal || 0).toFixed(1)} GB`}
          color={colorByPct(ramPct)} />
        <TopStat label="Диск" value={`${(node.diskUsage || 0).toFixed(1)}%`} color={colorByPct(node.diskUsage)} />
        <TopStat label="Сеть ↓" value={fmtBytes(node.netRecvSec)} color="text-cyan-400" />
        <TopStat label="Сеть ↑" value={fmtBytes(node.netSentSec)} color="text-blue-400" />
        <TopStat label="TCP" value={node.tcpEstablished || '—'}
          sub={node.tcpTotal > 0 ? `всего ${node.tcpTotal}` : undefined}
          color="text-violet-400" />
        <TopStat label="Процессов" value={node.processCount || '—'} color="text-slate-300"
          sub={node.loggedUsers > 0 ? `${node.loggedUsers} польз.` : undefined} />
        {(node.cpuTemp || 0) > 0
          ? <TopStat label="Темп. CPU" value={`${Math.round(node.cpuTemp)}°C`}
              color={node.cpuTemp > 80 ? 'text-red-400' : node.cpuTemp > 60 ? 'text-amber-400' : 'text-emerald-400'} />
          : <TopStat label="Аптайм" value={node.uptime || '—'} color="text-slate-400" />
        }
      </div>

      {/* ── Основная сетка ── */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-5">

        {/* ── CPU — широкий блок ── */}
        <Card title="Процессор (CPU)" icon={Cpu} iconColor="text-blue-400" className="xl:col-span-2">
          <div className="flex flex-col md:flex-row gap-6">
            {/* Gauge + метрики */}
            <div className="flex flex-col items-center gap-3 md:w-52 flex-shrink-0">
              <CpuGauge value={node.cpu} />
              <div className="w-full space-y-1">
                <InfoRow label="User" value={`${(node.cpuUser || 0).toFixed(1)}%`} />
                <InfoRow label="System" value={`${(node.cpuSystem || 0).toFixed(1)}%`} />
                {(node.cpuIowait || 0) > 0 && (
                  <InfoRow label="I/O Wait" value={`${(node.cpuIowait || 0).toFixed(1)}%`} />
                )}
                {(node.cpuSteal || 0) > 0 && (
                  <InfoRow label="Steal" value={`${(node.cpuSteal || 0).toFixed(1)}%`} />
                )}
                {node.cpuFreqMHz > 0 && (
                  <InfoRow label="Частота" value={`${(node.cpuFreqMHz / 1000).toFixed(2)} GHz`} />
                )}
                {(node.cpuTemp || 0) > 0 && (
                  <InfoRow label="Температура"
                    value={<span className={node.cpuTemp > 80 ? 'text-red-400' : node.cpuTemp > 60 ? 'text-amber-400' : 'text-emerald-400'}>
                      {Math.round(node.cpuTemp)}°C
                    </span>} />
                )}
                {!isWindows && node.loadAvg1 > 0 && (
                  <>
                    <InfoRow label="Load 1m" value={(node.loadAvg1 || 0).toFixed(2)} />
                    <InfoRow label="Load 5m" value={(node.loadAvg5 || 0).toFixed(2)} />
                    <InfoRow label="Load 15m" value={(node.loadAvg15 || 0).toFixed(2)} />
                  </>
                )}
              </div>
            </div>

            {/* История + breakdown + ядра */}
            <div className="flex-1 min-w-0 space-y-4">
              <CpuHistory data={node.cpuHistory || []} />

              {/* CPU breakdown stacked bar */}
              {(node.cpuUser > 0 || node.cpuSystem > 0) && (
                <div>
                  <div className="text-xs text-slate-500 font-medium mb-2 uppercase tracking-wider">Разбивка нагрузки</div>
                  <div className="w-full h-4 rounded-full overflow-hidden flex">
                    {node.cpuUser > 0 && (
                      <div className="h-full bg-blue-500 transition-all" style={{ width: `${node.cpuUser}%` }}
                        title={`User: ${node.cpuUser.toFixed(1)}%`} />
                    )}
                    {node.cpuSystem > 0 && (
                      <div className="h-full bg-violet-500 transition-all" style={{ width: `${node.cpuSystem}%` }}
                        title={`System: ${node.cpuSystem.toFixed(1)}%`} />
                    )}
                    {(node.cpuIowait || 0) > 0 && (
                      <div className="h-full bg-amber-500 transition-all" style={{ width: `${node.cpuIowait}%` }}
                        title={`I/O Wait: ${node.cpuIowait.toFixed(1)}%`} />
                    )}
                    {(node.cpuSteal || 0) > 0 && (
                      <div className="h-full bg-red-500 transition-all" style={{ width: `${node.cpuSteal}%` }}
                        title={`Steal: ${node.cpuSteal.toFixed(1)}%`} />
                    )}
                    <div className="h-full bg-slate-700/40 flex-1" />
                  </div>
                  <div className="flex gap-3 mt-1.5 flex-wrap">
                    {[
                      { label: 'User', value: node.cpuUser, color: 'bg-blue-500' },
                      { label: 'System', value: node.cpuSystem, color: 'bg-violet-500' },
                      { label: 'I/O Wait', value: node.cpuIowait, color: 'bg-amber-500' },
                      { label: 'Steal', value: node.cpuSteal, color: 'bg-red-500' },
                    ].filter(s => (s.value || 0) > 0).map(s => (
                      <div key={s.label} className="flex items-center gap-1 text-xs text-slate-500">
                        <span className={`w-2 h-2 rounded-sm ${s.color}`} />
                        {s.label}: <span className="text-slate-300 tabular-nums">{(s.value || 0).toFixed(1)}%</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* CPU по ядрам */}
              {cpuCores.length > 0 && (
                <div>
                  <div className="text-xs text-slate-500 font-medium mb-2 uppercase tracking-wider">
                    Загрузка по ядрам ({cpuCores.length})
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                    {cpuCores.map((pct, i) => (
                      <div key={i} className="bg-slate-900/60 rounded-lg p-2 border border-slate-700/30">
                        <div className="flex justify-between items-center mb-1">
                          <span className="text-xs text-slate-500">Core {i + 1}</span>
                          <span className={`text-xs font-bold tabular-nums ${colorByPct(pct)}`}>{pct.toFixed(0)}%</span>
                        </div>
                        <div className="w-full bg-slate-700/60 h-1.5 rounded-full overflow-hidden">
                          <div className={`h-full rounded-full transition-all duration-500 ${barColor(pct)}`}
                            style={{ width: `${Math.min(pct, 100)}%` }} />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </Card>

        {/* ── RAM + Накопители ── */}
        <Card title="Память и накопители" icon={Database} iconColor="text-violet-400">
          {/* RAM */}
          <div className="space-y-3 mb-4">
            <div className="flex items-center gap-1.5">
              <MemoryStick className="w-3.5 h-3.5 text-violet-400" />
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Оперативная память</span>
            </div>
            <div>
              <div className="flex justify-between text-xs mb-2">
                <span className="text-slate-400">Использовано</span>
                <span className="text-slate-200 font-medium tabular-nums">
                  {(node.ramUsed || 0).toFixed(1)} / {(node.ramTotal || 0).toFixed(1)} GB
                </span>
              </div>
              <ProgressBar value={ramPct} />
            </div>
            <div className="space-y-1">
              <InfoRow label="Свободно"
                value={`${Math.max(0, (node.ramTotal || 0) - (node.ramUsed || 0)).toFixed(2)} GB`} />
              {(node.ramCached || 0) > 0.01 && (
                <InfoRow label="Кэш" value={`${(node.ramCached || 0).toFixed(2)} GB`} />
              )}
              {(node.ramBuffers || 0) > 0.01 && (
                <InfoRow label="Буферы" value={`${(node.ramBuffers || 0).toFixed(2)} GB`} />
              )}
            </div>
            {(node.swapTotal || 0) > 0 && (
              <div className="pt-2 border-t border-slate-700/40">
                <div className="flex justify-between text-xs mb-2">
                  <span className="text-slate-400">Swap / Файл подкачки</span>
                  <span className="text-slate-400 tabular-nums">
                    {(node.swapUsed || 0).toFixed(1)} / {(node.swapTotal || 0).toFixed(1)} GB
                    <span className={`ml-1.5 font-medium ${colorByPct(swapPct)}`}>
                      {swapPct.toFixed(0)}%
                    </span>
                  </span>
                </div>
                <ProgressBar value={swapPct} />
              </div>
            )}

            {/* RAM history */}
            {node.ramHistory && node.ramHistory.length > 1 && (
              <div className="pt-2 border-t border-slate-700/40">
                <div className="text-xs text-slate-500 font-medium uppercase tracking-wider mb-2">История RAM</div>
                <ResponsiveContainer width="100%" height={60}>
                  <AreaChart data={node.ramHistory} margin={{ top: 2, right: 0, bottom: 0, left: 0 }}>
                    <defs>
                      <linearGradient id="ramGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="time" hide />
                    <Tooltip
                      contentStyle={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 8, fontSize: 11 }}
                      formatter={v => [`${v}%`, 'RAM']}
                      labelStyle={{ color: '#64748b' }}
                    />
                    <Area type="monotone" dataKey="value" stroke="#8b5cf6" strokeWidth={1.5}
                      fill="url(#ramGrad)" dot={false} isAnimationActive={false} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            )}
          </div>

          {/* Disk */}
          <div className="pt-3 border-t border-slate-700/40 space-y-3">
            <div className="flex items-center gap-1.5">
              <HardDrive className="w-3.5 h-3.5 text-amber-400" />
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Накопители</span>
            </div>
            {node.disks && node.disks.length > 0 ? (
              <div className="space-y-3">
                {node.disks.map(d => (
                  <div key={d.mount}>
                    <div className="flex justify-between text-xs mb-1">
                      <span className="text-slate-400 font-medium">{d.mount}</span>
                      <span className={`tabular-nums ${colorByPct(d.usedPercent)}`}>
                        {d.usedGB.toFixed(1)} / {d.totalGB.toFixed(1)} GB
                        <span className="text-slate-500 ml-1">({d.usedPercent.toFixed(1)}%)</span>
                      </span>
                    </div>
                    <div className="w-full bg-slate-700/60 h-2 rounded-full overflow-hidden">
                      <div className={`h-full rounded-full transition-all duration-500 ${barColor(d.usedPercent)}`}
                        style={{ width: `${Math.min(d.usedPercent, 100)}%` }} />
                    </div>
                    {d.fstype && <div className="text-xs text-slate-600 mt-0.5">{d.fstype}</div>}
                  </div>
                ))}
              </div>
            ) : (
              <div>
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-slate-400">{isWindows ? 'C:\\' : '/'}</span>
                  <span className={`tabular-nums ${colorByPct(node.diskUsage)}`}>
                    {(node.diskUsage || 0).toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-slate-700/60 h-2 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full ${barColor(node.diskUsage)}`}
                    style={{ width: `${Math.min(node.diskUsage || 0, 100)}%` }} />
                </div>
              </div>
            )}

            {/* Disk I/O */}
            {(node.diskReadSec > 0 || node.diskWriteSec > 0) && (
              <div className="pt-2 border-t border-slate-700/30">
                <div className="flex gap-2">
                  <div className="flex-1 bg-slate-900/50 rounded-lg p-2 border border-slate-700/30 text-center">
                    <div className="text-amber-400 font-bold text-sm tabular-nums">{fmtBytes(node.diskReadSec)}</div>
                    <div className="text-xs text-slate-600 mt-0.5">Чтение/с</div>
                  </div>
                  <div className="flex-1 bg-slate-900/50 rounded-lg p-2 border border-slate-700/30 text-center">
                    <div className="text-orange-400 font-bold text-sm tabular-nums">{fmtBytes(node.diskWriteSec)}</div>
                    <div className="text-xs text-slate-600 mt-0.5">Запись/с</div>
                  </div>
                  {(node.diskQueue || 0) > 0 && (
                    <div className="flex-1 bg-slate-900/50 rounded-lg p-2 border border-slate-700/30 text-center">
                      <div className={`font-bold text-sm tabular-nums ${node.diskQueue > 2 ? 'text-red-400' : node.diskQueue > 1 ? 'text-amber-400' : 'text-slate-300'}`}>
                        {(node.diskQueue || 0).toFixed(2)}
                      </div>
                      <div className="text-xs text-slate-600 mt-0.5">Очередь</div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* ── Сеть ── */}
        <Card title={`Сеть${node.netInterface ? ` · ${node.netInterface}` : ''}`}
              icon={Globe} iconColor="text-cyan-400" className="xl:col-span-2">
          <div className="flex gap-3 mb-4">
            <div className="flex-1 bg-slate-900/60 rounded-xl p-3 text-center border border-slate-700/40">
              <div className="text-cyan-400 font-bold text-lg tabular-nums">↓ {fmtBytes(node.netRecvSec)}</div>
              <div className="text-xs text-slate-500 mt-0.5">Входящий трафик</div>
            </div>
            <div className="flex-1 bg-slate-900/60 rounded-xl p-3 text-center border border-slate-700/40">
              <div className="text-blue-400 font-bold text-lg tabular-nums">↑ {fmtBytes(node.netSentSec)}</div>
              <div className="text-xs text-slate-500 mt-0.5">Исходящий трафик</div>
            </div>
          </div>
          <NetworkLines data={node.netHistory || []} />

          {/* Все интерфейсы */}
          {node.allIfaces && node.allIfaces.length > 1 && (
            <div className="mt-4 pt-4 border-t border-slate-700/40">
              <div className="text-xs text-slate-500 font-medium uppercase tracking-wider mb-2">
                Все интерфейсы
              </div>
              <div className="space-y-2">
                {node.allIfaces.map(iface => (
                  <div key={iface.name}
                    className="flex items-center justify-between text-xs py-1.5 border-b border-slate-700/20 last:border-0">
                    <span className="text-slate-400 font-medium flex items-center gap-1.5">
                      <Wifi className="w-3 h-3 text-slate-500" />{iface.name}
                    </span>
                    <span className="text-slate-500 tabular-nums">
                      ↓{fmtBytes(iface.bytesRecvSec)} · ↑{fmtBytes(iface.bytesSentSec)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Card>

        {/* ── Сервисы ── */}
        <Card title="Сервисы" icon={Shield} iconColor="text-emerald-400">
          <div className="space-y-2">
            <ServiceBadge label="SSH" port={22} active={node.sshReachable} ms={node.sshMs} />
            {isWindows && (
              <>
                <ServiceBadge label="Remote Desktop (RDP)" port={3389} active={node.rdpReachable} ms={node.rdpMs} />
                <ServiceBadge label="File Sharing (SMB)" port={445} active={node.smbReachable} ms={node.smbMs} />
                <ServiceBadge label="WinRM" port={5985} active={node.winrmReachable} ms={node.winrmMs} />
              </>
            )}
            <ServiceBadge label="HTTP" port={80} active={node.httpReachable} ms={node.httpMs} />
            <ServiceBadge label="DNS" port={53} active={node.dnsReachable} ms={node.dnsMs} />
          </div>
          {(node.tcpTotal || 0) > 0 && (
            <div className="mt-4 pt-4 border-t border-slate-700/40 space-y-1">
              <div className="text-xs text-slate-500 font-medium uppercase tracking-wider mb-2">TCP-соединения</div>
              <InfoRow label="Established" value={node.tcpEstablished || 0} />
              <InfoRow label="Time Wait" value={node.tcpTimeWait || 0} />
              <InfoRow label="Всего" value={node.tcpTotal || 0} />
            </div>
          )}
        </Card>

        {/* ── Топ процессов (CPU / RAM) ── */}
        <ProcessesCard node={node} className="xl:col-span-2" />

        {/* ── Системная информация ── */}
        <Card title="Система" icon={TrendingUp} iconColor="text-slate-400">
          <div className="space-y-1">
            <InfoRow label="Имя хоста" value={node.name} />
            <InfoRow label="IP-адрес" value={node.ip || '—'} />
            <InfoRow label="ОС" value={getOSLabel(node.os)} />
            {node.cpuModel && <InfoRow label="Процессор" value={node.cpuModel} />}
            {node.cpuFreqMHz > 0 && (
              <InfoRow label="Частота CPU" value={`${(node.cpuFreqMHz / 1000).toFixed(2)} GHz`} />
            )}
            <InfoRow label="Аптайм" value={node.uptime || '—'} />
            {node.bootTime && <InfoRow label="Последняя загрузка" value={node.bootTime} />}
            {node.loggedUsers > 0 && (
              <InfoRow label="Пользователей онлайн" value={node.loggedUsers} />
            )}
            <InfoRow label="Всего процессов" value={node.processCount || '—'} />
          </div>
        </Card>

        {/* ── SNMP ── (показываем если собраны данные) */}
        {node.snmpCollected && (
          <Card title="SNMP" icon={Activity} iconColor="text-cyan-400">
            <div className="space-y-1">
              {node.snmpSysName && <InfoRow label="Системное имя" value={node.snmpSysName} />}
              {node.snmpSysUpTime > 0 && (
                <InfoRow label="Аптайм (SNMP)" value={`${Math.floor(node.snmpSysUpTime / 3600)} ч. ${Math.floor((node.snmpSysUpTime % 3600) / 60)} мин.`} />
              )}
              {node.snmpCpuLoad > 0 && (
                <InfoRow label="CPU Load (hrProcessorLoad)" value={`${node.snmpCpuLoad}%`} />
              )}
              {node.snmpIfCount > 0 && (
                <InfoRow label="Интерфейсов (ifNumber)" value={node.snmpIfCount} />
              )}
            </div>
          </Card>
        )}

        {/* ── FSRM ── (только Windows/srv-corp-01) */}
        {node.fsrm && node.fsrm.length > 0 && (
          <Card title="FSRM — Квоты" icon={HardDrive} iconColor="text-orange-400" className="xl:col-span-2">
            <div className="space-y-4">
              {node.fsrm.map((q, i) => {
                const pct = q.quotaUsedPercent || 0;
                return (
                  <div key={i} className="bg-slate-900/50 rounded-xl p-3 border border-slate-700/30">
                    <div className="flex justify-between items-center mb-2">
                      <span className="text-sm font-medium text-slate-200">{q.quotaPath || 'Unknown'}</span>
                      <span className={`text-xs font-bold tabular-nums ${colorByPct(pct)}`}>{pct.toFixed(1)}%</span>
                    </div>
                    <div className="w-full bg-slate-700/60 h-2 rounded-full overflow-hidden mb-2">
                      <div className={`h-full rounded-full transition-all ${barColor(pct)}`}
                        style={{ width: `${Math.min(pct, 100)}%` }} />
                    </div>
                    <div className="flex justify-between text-xs text-slate-500">
                      <span>
                        {(q.quotaUsedBytes / 1024 / 1024 / 1024).toFixed(2)} GB /&nbsp;
                        {(q.quotaLimitBytes / 1024 / 1024 / 1024).toFixed(2)} GB
                      </span>
                      {q.violations24h > 0 && (
                        <span className="text-red-400 font-medium">⚠ {q.violations24h} нарушений за 24 ч</span>
                      )}
                    </div>
                    {q.lastViolationTime && (
                      <div className="text-xs text-slate-600 mt-1">
                        Последнее: {q.lastViolationTime} — {q.lastViolationType}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </Card>
        )}

      </div>
    </div>
  );
}

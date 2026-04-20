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

function ServiceBadge({ label, port, active }) {
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
      <span className={`text-xs font-semibold px-2 py-0.5 rounded-md
        ${active ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
        {active ? 'Running' : 'Stopped'}
      </span>
    </div>
  );
}

// Таблица процессов — вынесена в компонент для переиспользования
function ProcessTable({ processes, emptyText = 'Нет данных' }) {
  if (!processes || processes.length === 0) {
    return <div className="text-slate-500 text-sm py-4 text-center">{emptyText}</div>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="text-slate-500 uppercase">
            <th className="text-left pb-2 pr-3 font-medium">PID</th>
            <th className="text-left pb-2 pr-3 font-medium">Процесс</th>
            <th className="text-right pb-2 pr-3 font-medium">CPU %</th>
            <th className="text-right pb-2 pr-3 font-medium">RAM</th>
            <th className="text-left pb-2 font-medium hidden sm:table-cell">Пользователь</th>
          </tr>
        </thead>
        <tbody>
          {processes.map(proc => (
            <tr key={proc.pid} className="border-t border-slate-700/30 hover:bg-slate-700/20 transition-colors">
              <td className="py-2 pr-3 text-slate-500 tabular-nums">{proc.pid}</td>
              <td className="py-2 pr-3 font-medium text-slate-200 max-w-[140px] truncate">{proc.name}</td>
              <td className="py-2 pr-3 text-right">
                <div className="flex items-center justify-end gap-2">
                  <div className="w-10 bg-slate-700 h-1 rounded-full overflow-hidden hidden sm:block">
                    <div className={`h-full rounded-full ${barColor(proc.cpu)}`}
                      style={{ width: `${Math.min(proc.cpu, 100)}%` }} />
                  </div>
                  <span className={`tabular-nums w-10 text-right ${colorByPct(proc.cpu)}`}>
                    {proc.cpu.toFixed(1)}
                  </span>
                </div>
              </td>
              <td className="py-2 pr-3 text-right text-slate-400 tabular-nums">
                {proc.ram >= 1024 ? `${(proc.ram / 1024).toFixed(1)} GB` : `${Math.round(proc.ram)} MB`}
              </td>
              <td className="py-2 text-slate-500 hidden sm:table-cell truncate max-w-[120px]">
                {proc.user || '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Карточка процессов с переключателем CPU / RAM
function ProcessesCard({ node, className }) {
  const [tab, setTab] = useState('cpu');
  const processes = tab === 'cpu' ? node.processes : node.topMemProcesses;

  return (
    <Card title="Топ процессов" icon={List} iconColor="text-slate-400" className={className}>
      {/* Табы */}
      <div className="flex gap-1 mb-4 bg-slate-900/50 rounded-lg p-1 w-fit">
        {[{ key: 'cpu', label: 'По CPU' }, { key: 'ram', label: 'По RAM' }].map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-all duration-200
              ${tab === t.key
                ? 'bg-slate-700 text-slate-100 shadow'
                : 'text-slate-500 hover:text-slate-300'}`}
          >
            {t.label}
          </button>
        ))}
      </div>
      <ProcessTable processes={processes} emptyText="Данные о процессах ещё не поступали" />
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
      <div className="grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-6 gap-3 mb-6">
        <TopStat label="CPU" value={`${node.cpu}%`} color={colorByPct(node.cpu)} />
        <TopStat label="RAM" value={`${ramPct.toFixed(0)}%`}
          sub={`${(node.ramUsed || 0).toFixed(1)} / ${(node.ramTotal || 0).toFixed(1)} GB`}
          color={colorByPct(ramPct)} />
        <TopStat label="Диск" value={`${(node.diskUsage || 0).toFixed(1)}%`} color={colorByPct(node.diskUsage)} />
        <TopStat label="Сеть ↓" value={fmtBytes(node.netRecvSec)} color="text-cyan-400" />
        <TopStat label="Сеть ↑" value={fmtBytes(node.netSentSec)} color="text-blue-400" />
        <TopStat label="Процессов" value={node.processCount || '—'} color="text-slate-300"
          sub={node.loggedUsers > 0 ? `${node.loggedUsers} польз.` : undefined} />
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
                {node.cpuFreqMHz > 0 && (
                  <InfoRow label="Частота" value={`${(node.cpuFreqMHz / 1000).toFixed(2)} GHz`} />
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

            {/* История + ядра */}
            <div className="flex-1 min-w-0 space-y-4">
              <CpuHistory data={node.cpuHistory || []} />

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
                  </span>
                </div>
                <ProgressBar value={swapPct} />
              </div>
            )}
          </div>

          {/* Disk */}
          <div className="pt-3 border-t border-slate-700/40 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5">
                <HardDrive className="w-3.5 h-3.5 text-amber-400" />
                <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Накопители</span>
              </div>
              {(node.diskReadSec > 0 || node.diskWriteSec > 0) && (
                <div className="flex gap-2 text-xs">
                  <span className="text-amber-400 tabular-nums">↑{fmtBytes(node.diskReadSec)}</span>
                  <span className="text-orange-400 tabular-nums">↓{fmtBytes(node.diskWriteSec)}</span>
                </div>
              )}
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
            {isWindows ? (
              <>
                <ServiceBadge label="Remote Desktop (RDP)" port={3389} active={node.rdpRunning} />
                <ServiceBadge label="File Sharing (SMB)" port={445} active={node.smbRunning} />
              </>
            ) : (
              <ServiceBadge label="SSH" port={22} active={node.online} />
            )}
          </div>
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

      </div>
    </div>
  );
}

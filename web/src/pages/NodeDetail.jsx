import { useParams, Link } from 'react-router-dom';
import {
  ArrowLeft, Cpu, Database, HardDrive, Globe,
  Activity, Monitor, Terminal, List, Shield, TrendingUp
} from 'lucide-react';
import { useNodes } from '../hooks/useNodes';
import CpuGauge from '../components/charts/CpuGauge';
import CpuHistory from '../components/charts/CpuHistory';
import DiskBars from '../components/charts/DiskBars';
import NetworkLines from '../components/charts/NetworkLines';
import ProgressBar from '../components/common/ProgressBar';

// ─── Утилиты ───────────────────────────────────────────────────────────────

function formatBytes(bytes) {
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
    <div className={`bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5 ${className}`}>
      <div className="flex items-center gap-2 mb-4">
        <Icon className={`w-4 h-4 ${iconColor}`} />
        <span className="text-sm font-semibold text-slate-300">{title}</span>
      </div>
      {children}
    </div>
  );
}

function TopStat({ label, value, color = 'text-slate-100' }) {
  return (
    <div className="bg-slate-900/50 border border-slate-700/40 rounded-xl p-3 text-center">
      <div className={`text-lg font-bold tabular-nums ${color}`}>{value}</div>
      <div className="text-xs text-slate-500 mt-0.5">{label}</div>
    </div>
  );
}

function InfoRow({ label, value }) {
  return (
    <div className="flex justify-between items-center py-1.5 border-b border-slate-700/30 last:border-0">
      <span className="text-xs text-slate-500">{label}</span>
      <span className="text-xs text-slate-300 font-medium tabular-nums">{value}</span>
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

// ─── Главный компонент ─────────────────────────────────────────────────────

export default function NodeDetail() {
  const { nodeId } = useParams();
  const { data: nodes, loading, error } = useNodes();

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
        Узел <span className="text-slate-200 font-medium">{nodeId}</span> не найден или отключён.
      </div>
    );
  }

  const isWindows = node.os?.toLowerCase().includes('windows');
  const ramPct = node.ramTotal > 0 ? (node.ramUsed / node.ramTotal) * 100 : 0;
  const swapPct = node.swapTotal > 0 ? (node.swapUsed / node.swapTotal) * 100 : 0;

  // История CPU → формат для графика
  const cpuHistoryData = (node.cpuHistory || []).map((p, i) => ({
    time: `-${(node.cpuHistory.length - i) * 3}с`,
    cpu: p.value,
  }));

  // История сети → формат для графика (байт/сек)
  const netHistoryData = (node.netHistory || []).map((p, i) => ({
    time: `-${(node.netHistory.length - i) * 3}с`,
    recv: p.recv,
    sent: p.sent,
  }));

  // Диск
  const diskData = [{ mount: isWindows ? 'C:\\' : '/', used: node.diskUsage || 0, total: 100, unit: '%' }];

  return (
    <div className="p-6 max-w-screen-2xl mx-auto">

      {/* ── Хлебные крошки + заголовок ── */}
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
              : <Terminal className="w-5 h-5 text-emerald-400" />
            }
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-bold text-slate-100">{node.name}</h1>
              <span className={`w-2 h-2 rounded-full ${node.online ? 'bg-emerald-400' : 'bg-red-400'}`} />
              <span className={`text-xs px-2 py-0.5 rounded-full font-medium
                ${node.online ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
                {node.online ? 'Online' : 'Offline'}
              </span>
            </div>
            <p className="text-slate-500 text-sm">
              {getOSLabel(node.os)} · {node.ip || '—'} · ⬆ {node.uptime || '0 ч.'}
            </p>
          </div>
        </div>
      </div>

      {/* ── Мини-статистика (4 быстрых показателя) ── */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
        <TopStat label="CPU" value={`${node.cpu}%`} color={colorByPct(node.cpu)} />
        <TopStat label="RAM" value={`${ramPct.toFixed(0)}%`} color={colorByPct(ramPct)} />
        <TopStat label="Диск" value={`${(node.diskUsage || 0).toFixed(1)}%`} color={colorByPct(node.diskUsage)} />
        <TopStat
          label={node.netInterface || 'Сеть'}
          value={formatBytes(node.netRecvSec)}
          color="text-cyan-400"
        />
      </div>

      {/* ── Основная сетка ── */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-5">

        {/* CPU — большой блок */}
        <Card title="Процессор (CPU)" icon={Cpu} iconColor="text-blue-400" className="xl:col-span-2">
          <div className="flex flex-col md:flex-row gap-6">
            {/* Gauge + разбивка */}
            <div className="flex flex-col items-center gap-4 md:w-52 flex-shrink-0">
              <CpuGauge value={node.cpu} />
              <div className="w-full space-y-1">
                <InfoRow label="User" value={`${(node.cpuUser || 0).toFixed(1)}%`} />
                <InfoRow label="System" value={`${(node.cpuSystem || 0).toFixed(1)}%`} />
                {!isWindows && (
                  <>
                    <InfoRow label="Load 1m" value={(node.loadAvg1 || 0).toFixed(2)} />
                    <InfoRow label="Load 5m" value={(node.loadAvg5 || 0).toFixed(2)} />
                    <InfoRow label="Load 15m" value={(node.loadAvg15 || 0).toFixed(2)} />
                  </>
                )}
              </div>
            </div>
            {/* История */}
            <div className="flex-1 min-w-0">
              <CpuHistory data={cpuHistoryData} />
            </div>
          </div>
        </Card>

        {/* RAM */}
        <Card title="Оперативная память" icon={Database} iconColor="text-violet-400">
          <div className="space-y-4">
            <div>
              <div className="flex justify-between text-xs mb-2">
                <span className="text-slate-400">Использовано</span>
                <span className="text-slate-200 font-medium tabular-nums">
                  {(node.ramUsed || 0).toFixed(1)} / {(node.ramTotal || 0).toFixed(1)} GB
                </span>
              </div>
              <ProgressBar value={ramPct} />
            </div>

            <div className="space-y-1 pt-1">
              <InfoRow label="Используется" value={`${(node.ramUsed || 0).toFixed(2)} GB`} />
              <InfoRow label="Свободно" value={`${Math.max(0, (node.ramTotal || 0) - (node.ramUsed || 0) - (node.ramCached || 0)).toFixed(2)} GB`} />
              {(node.ramCached || 0) > 0 && (
                <InfoRow label="Кэш" value={`${(node.ramCached || 0).toFixed(2)} GB`} />
              )}
              {(node.ramBuffers || 0) > 0 && (
                <InfoRow label="Буферы" value={`${(node.ramBuffers || 0).toFixed(2)} GB`} />
              )}
            </div>

            {(node.swapTotal || 0) > 0 && (
              <div className="pt-2 border-t border-slate-700/40">
                <div className="flex justify-between text-xs mb-2">
                  <span className="text-slate-400">Swap</span>
                  <span className="text-slate-400 tabular-nums">
                    {(node.swapUsed || 0).toFixed(1)} / {(node.swapTotal || 0).toFixed(1)} GB
                  </span>
                </div>
                <ProgressBar value={swapPct} />
              </div>
            )}
          </div>
        </Card>

        {/* Сеть */}
        <Card title={`Сеть${node.netInterface ? ` · ${node.netInterface}` : ''}`}
              icon={Globe} iconColor="text-cyan-400" className="xl:col-span-2">
          <div className="flex gap-3 mb-4">
            <div className="flex-1 bg-slate-900/60 rounded-xl p-3 text-center border border-slate-700/40">
              <div className="text-cyan-400 font-bold text-lg">↓ {formatBytes(node.netRecvSec)}</div>
              <div className="text-xs text-slate-500 mt-0.5">Входящий</div>
            </div>
            <div className="flex-1 bg-slate-900/60 rounded-xl p-3 text-center border border-slate-700/40">
              <div className="text-blue-400 font-bold text-lg">↑ {formatBytes(node.netSentSec)}</div>
              <div className="text-xs text-slate-500 mt-0.5">Исходящий</div>
            </div>
          </div>
          <NetworkLines data={netHistoryData} />
        </Card>

        {/* Диски */}
        <Card title="Накопители" icon={HardDrive} iconColor="text-amber-400">
          <DiskBars disks={diskData} />
        </Card>

        {/* Процессы */}
        <Card title="Топ процессов по CPU" icon={List} iconColor="text-slate-400" className="xl:col-span-2">
          {!node.processes || node.processes.length === 0 ? (
            <div className="text-slate-500 text-sm py-4 text-center">
              Данные о процессах ещё не поступали
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-slate-500 uppercase">
                    <th className="text-left pb-2 pr-4 font-medium">PID</th>
                    <th className="text-left pb-2 pr-4 font-medium">Процесс</th>
                    <th className="text-right pb-2 pr-4 font-medium">CPU %</th>
                    <th className="text-right pb-2 pr-4 font-medium">RAM</th>
                    <th className="text-left pb-2 font-medium hidden sm:table-cell">Пользователь</th>
                  </tr>
                </thead>
                <tbody>
                  {node.processes.map((proc, idx) => (
                    <tr key={proc.pid} className="border-t border-slate-700/30 hover:bg-slate-700/20 transition-colors">
                      <td className="py-2 pr-4 text-slate-500 tabular-nums">{proc.pid}</td>
                      <td className="py-2 pr-4 font-medium text-slate-200">{proc.name}</td>
                      <td className="py-2 pr-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <div className="w-12 bg-slate-700 h-1 rounded-full overflow-hidden hidden sm:block">
                            <div
                              className={`h-full rounded-full ${proc.cpu > 50 ? 'bg-red-500' : proc.cpu > 20 ? 'bg-amber-500' : 'bg-blue-500'}`}
                              style={{ width: `${Math.min(proc.cpu, 100)}%` }}
                            />
                          </div>
                          <span className={`tabular-nums w-10 text-right ${colorByPct(proc.cpu)}`}>
                            {proc.cpu.toFixed(1)}
                          </span>
                        </div>
                      </td>
                      <td className="py-2 pr-4 text-right text-slate-400 tabular-nums">
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
          )}
        </Card>

        {/* Службы */}
        <Card title="Сервисы" icon={Shield} iconColor="text-emerald-400">
          <div className="space-y-2">
            {isWindows ? (
              <>
                <ServiceBadge label="Remote Desktop (RDP)" port={3389} active={node.rdpRunning} />
                <ServiceBadge label="File Sharing (SMB)" port={445} active={node.smbRunning} />
              </>
            ) : (
              <>
                <ServiceBadge label="SSH" port={22} active={node.online} />
              </>
            )}
          </div>

          {/* Системная информация */}
          <div className="mt-5 pt-4 border-t border-slate-700/40">
            <div className="flex items-center gap-2 mb-3">
              <TrendingUp className="w-3.5 h-3.5 text-slate-500" />
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Система</span>
            </div>
            <div className="space-y-1">
              <InfoRow label="Имя" value={node.name} />
              <InfoRow label="IP-адрес" value={node.ip || '—'} />
              <InfoRow label="Аптайм" value={node.uptime || '—'} />
              <InfoRow label="Ping" value={`${node.ping || 1} мс`} />
            </div>
          </div>
        </Card>

      </div>
    </div>
  );
}

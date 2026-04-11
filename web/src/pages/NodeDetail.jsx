import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Cpu, Activity, Database, HardDrive, Globe } from 'lucide-react';
import MetricCard from '../components/common/MetricCard';
import ProgressBar from '../components/common/ProgressBar';
import CpuHistory from '../components/charts/CpuHistory';
import CpuGauge from '../components/charts/CpuGauge';
import RamPie from '../components/charts/RamPie';
import RamBar from '../components/charts/RamBar';
import DiskBars from '../components/charts/DiskBars';
import NetworkLines from '../components/charts/NetworkLines';
import { List, ShieldAlert } from 'lucide-react';
import ProcessTable from '../components/tables/ProcessTable';
import ServiceStatus from '../components/status/ServiceStatus';
import FsrmQuota from '../components/status/FsrmQuota';

// Генераторы фейковых данных для графиков
const generateHistory = () => {
  const now = new Date();
  return Array.from({ length: 15 }, (_, i) => {
    const d = new Date(now.getTime() - (14 - i) * 60000);
    return {
      time: `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')}`,
      cpu: Math.floor(Math.random() * 40) + 30,
      rx: Math.floor(Math.random() * 1000) + 200,
      tx: Math.floor(Math.random() * 500) + 50
    };
  });
};

export default function NodeDetail() {
  const { nodeId } = useParams();
  
  // Тестовые данные (Mock)
  const currentCpu = 68;
  const historyData = generateHistory();
  const ramData = { used: 2.1, cached: 0.8, free: 1.1, total: 4.0 };
  const diskData = [
    { mount: '/', used: 12, total: 40 },
    { mount: '/var/log', used: 3.5, total: 10 },
    { mount: '/home', used: 45, total: 100 }
  ];

  // Новые тестовые данные
  const mockProcesses = [
    { pid: 1234, name: 'nginx', cpu: 14.2, ram: 150 },
    { pid: 5678, name: 'go-agent', cpu: 2.1, ram: 22 },
    { pid: 9012, name: 'mysqld', cpu: 0.5, ram: 450 },
    { pid: 3456, name: 'sshd', cpu: 0.1, ram: 12 },
  ];

  const mockServices = [
    { name: 'TermService', displayName: 'Службы удаленных рабочих столов', state: 'Running' },
    { name: 'LanmanServer', displayName: 'Сервер (SMB)', state: 'Running' },
    { name: 'SNMP', displayName: 'Служба SNMP', state: 'Stopped' },
  ];

  return (
    <div className="p-6">
      {/* Навигация и заголовок */}
      <div className="mb-6 flex items-center gap-4">
        <Link to="/" className="p-2 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors border border-slate-700 text-slate-400 hover:text-slate-100">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            Узел: <span className="text-blue-400">{nodeId}</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">ОС: Astra Linux 1.7 · IP: 10.10.10.100 · Uptime: 3д 14ч</p>
        </div>
      </div>

      {/* Сетка метрик */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        
        {/* Секция CPU */}
        <MetricCard title="Процессор (CPU)" icon={Cpu} className="xl:col-span-3">
          <div className="flex flex-col md:flex-row gap-6 items-center">
            <div className="w-full md:w-1/4 flex flex-col items-center border-b md:border-b-0 md:border-r border-slate-700 pb-6 md:pb-0 pr-0 md:pr-6">
              <CpuGauge value={currentCpu} />
              <p className="text-slate-400 text-sm mt-2 text-center">4 Ядра (x86_64)</p>
            </div>
            <div className="w-full md:w-3/4">
              <CpuHistory data={historyData} />
            </div>
          </div>
        </MetricCard>

        {/* Секция RAM */}
        <MetricCard title="Оперативная память (RAM)" icon={Database} className="xl:col-span-1">
          
          <RamBar used={ramData.used} total={ramData.total} />
          
          <RamPie used={ramData.used} cached={ramData.cached} free={ramData.free} />
          {/* Легенда */}
          <div className="flex justify-center gap-4 text-xs mt-2">
            <div className="flex items-center gap-1.5"><div className="w-2 h-2 rounded-full bg-blue-500"/>Исп.</div>
            <div className="flex items-center gap-1.5"><div className="w-2 h-2 rounded-full bg-amber-500"/>Кэш</div>
            <div className="flex items-center gap-1.5"><div className="w-2 h-2 rounded-full bg-slate-600"/>Своб.</div>
          </div>
        </MetricCard>

        {/* Секция Дисков */}
        <MetricCard title="Накопители (Disks)" icon={HardDrive} className="xl:col-span-1">
          <DiskBars disks={diskData} />
        </MetricCard>

        {/* Секция Сети */}
        <MetricCard title="Сеть (Network: ens192)" icon={Globe} className="xl:col-span-1">
          <div className="flex justify-between items-center bg-slate-900/50 p-3 rounded-lg border border-slate-700/50 mb-2">
            <div className="text-green-500 font-medium text-sm flex items-center gap-2">↓ 1.2 MB/s</div>
            <div className="text-blue-500 font-medium text-sm flex items-center gap-2">↑ 340 KB/s</div>
          </div>
          <NetworkLines data={historyData} />
        </MetricCard>

        {/* Секция Процессов */}
        <MetricCard title="Топ процессов (по CPU)" icon={List} className="xl:col-span-2">
          <ProcessTable processes={mockProcesses} />
        </MetricCard>

        {/* Секция Windows-специфики (Службы и FSRM) */}
        {/* В реальном приложении мы бы скрывали её для Linux-узлов (node.os !== 'Windows') */}
        <MetricCard title="Службы и Квоты (Windows)" icon={ShieldAlert} className="xl:col-span-1">
          <div className="space-y-6">
            <div>
              <h3 className="text-xs text-slate-500 uppercase font-semibold mb-3">Состояние служб</h3>
              <ServiceStatus services={mockServices} />
            </div>
            <div>
              <h3 className="text-xs text-slate-500 uppercase font-semibold mb-3">FSRM Квоты (srv-corp)</h3>
              <FsrmQuota path="C:\CorpShare" usedGB={8.2} totalGB={10} thresholdPercent={80} />
            </div>
          </div>
        </MetricCard>

      </div>
    </div>
  );
}
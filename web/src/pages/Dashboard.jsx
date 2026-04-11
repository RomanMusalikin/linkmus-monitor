import NodeCard from '../components/cards/NodeCard';

// Генератор случайного графика для тестов
const generateMockHistory = () => Array.from({ length: 20 }, () => ({ value: Math.floor(Math.random() * 40) + 20 }));

// Тестовые данные (твои реальные машины из курсовой!)
const MOCK_NODES = [
  { name: 'srv-mon-01', os: 'Astra Linux 1.7', ip: '10.10.10.10', online: true, cpu: 12, ramUsed: 1024, ramTotal: 4096, uptime: '5д 2ч', ping: 1, cpuHistory: generateMockHistory() },
  { name: 'srv-corp-01', os: 'Windows Server', ip: '10.10.10.11', online: true, cpu: 88, ramUsed: 7168, ramTotal: 8192, uptime: '12д 8ч', ping: 3, cpuHistory: generateMockHistory() },
  { name: 'gw-border-01', os: 'MikroTik CHR', ip: '10.10.10.1', online: true, cpu: 5, ramUsed: 256, ramTotal: 1024, uptime: '30д 1ч', ping: 1, cpuHistory: generateMockHistory() },
  { name: 'cl-astra-01', os: 'Astra Linux 1.7', ip: '10.10.10.100', online: true, cpu: 73, ramUsed: 2148, ramTotal: 4096, uptime: '3д 14ч', ping: 2, cpuHistory: generateMockHistory() },
  { name: 'cl-win-01', os: 'Windows 11', ip: '10.10.10.101', online: true, cpu: 45, ramUsed: 4096, ramTotal: 8192, uptime: '1д 5ч', ping: 4, cpuHistory: generateMockHistory() },
  { name: 'cl-redos-01', os: 'РЕД ОС 7.3', ip: '10.10.10.102', online: false, cpu: 0, ramUsed: 0, ramTotal: 4096, uptime: 'Offline', ping: 0, cpuHistory: generateMockHistory().map(() => ({value: 0})) },
];

export default function Dashboard() {
  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-end">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 mb-1">Главная панель</h1>
          <p className="text-slate-400 text-sm">Обзор состояния узлов инфраструктуры</p>
        </div>
        <div className="text-sm text-slate-400 bg-slate-800 px-3 py-1.5 rounded-lg border border-slate-700">
          Онлайн: <span className="text-green-400 font-medium">5</span> / 6
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {MOCK_NODES.map(node => (
          <NodeCard key={node.name} node={node} />
        ))}
      </div>
    </div>
  );
}
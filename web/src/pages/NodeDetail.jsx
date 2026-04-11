import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Cpu, Activity } from 'lucide-react';
import MetricCard from '../components/common/MetricCard';
import CpuHistory from '../components/charts/CpuHistory';
import CpuGauge from '../components/charts/CpuGauge';

// Генератор фейковых данных для графика
const generateHistory = () => {
  const now = new Date();
  return Array.from({ length: 15 }, (_, i) => {
    const d = new Date(now.getTime() - (14 - i) * 60000); // поминутно
    return {
      time: `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')}`,
      cpu: Math.floor(Math.random() * 40) + 30 // случайные данные от 30 до 70
    };
  });
};

export default function NodeDetail() {
  const { nodeId } = useParams();
  
  // Пока хардкодим текущие значения для теста (позже это заменит хук useNodeDetail)
  const currentCpu = 68;
  const cpuHistoryData = generateHistory();

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
        
        {/* Секция CPU занимает всю ширину на мобилках и 3 колонки на больших экранах */}
        <MetricCard title="Процессор (CPU)" icon={Cpu} className="xl:col-span-3">
          <div className="flex flex-col md:flex-row gap-6 items-center">
            {/* Левая часть: Датчик (бублик) */}
            <div className="w-full md:w-1/4 flex flex-col items-center border-b md:border-b-0 md:border-r border-slate-700 pb-6 md:pb-0 pr-0 md:pr-6">
              <CpuGauge value={currentCpu} />
              <p className="text-slate-400 text-sm mt-2 text-center">4 Ядра (x86_64)</p>
            </div>
            
            {/* Правая часть: График */}
            <div className="w-full md:w-3/4">
              <div className="flex items-center gap-2 mb-2">
                <Activity className="w-4 h-4 text-slate-500" />
                <span className="text-slate-400 text-sm">История за последние 15 минут</span>
              </div>
              <CpuHistory data={cpuHistoryData} />
            </div>
          </div>
        </MetricCard>

        {/* Здесь потом будут карточки RAM, Дисков и Сети... */}
      </div>
    </div>
  );
}
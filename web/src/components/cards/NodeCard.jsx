import { Link } from 'react-router-dom';
import ProgressBar from '../common/ProgressBar';
import Sparkline from '../charts/Sparkline';

export default function NodeCard({ node }) {
  const statusColor = node.online ? 'bg-green-500' : 'bg-red-500';
  const cpuColor = node.cpu > 85 ? 'text-red-400' : node.cpu > 60 ? 'text-amber-400' : 'text-green-400';

  // Переводим МБ в ГБ для красивого отображения
  const ramUsedGB = (node.ramUsed / 1024).toFixed(1);
  const ramTotalGB = (node.ramTotal / 1024).toFixed(1);

  return (
    <Link 
      to={`/node/${node.name}`} 
      className="bg-slate-800 rounded-xl p-5 hover:bg-slate-750 transition-all border border-slate-700 hover:border-blue-500/50 block"
    >
      {/* Шапка карточки */}
      <div className="flex items-center gap-2 mb-2">
        <div className={`w-2.5 h-2.5 rounded-full ${statusColor}`} />
        <span className="text-slate-100 font-semibold">{node.name}</span>
      </div>
      <p className="text-slate-400 text-sm mb-5">{node.os} · {node.ip}</p>

      {/* Индикаторы */}
      <div className="space-y-4">
        <div>
          <div className="flex justify-between text-sm mb-1.5">
            <span className="text-slate-400">CPU</span>
            <span className={cpuColor}>{node.cpu}%</span>
          </div>
          <ProgressBar value={node.cpu} />
        </div>

        <div>
          <div className="flex justify-between text-sm mb-1.5">
            <span className="text-slate-400">RAM</span>
            <span className="text-slate-200">{ramUsedGB} / {ramTotalGB} GB</span>
          </div>
          <ProgressBar value={(node.ramUsed / node.ramTotal) * 100} />
        </div>
      </div>

      {/* График */}
      <div className="mt-5 h-10">
        <Sparkline data={node.cpuHistory} color={node.online ? "#3b82f6" : "#475569"} />
      </div>

      {/* Подвал */}
      <p className="text-slate-500 text-xs mt-4">
        Uptime: {node.uptime} · Ping: {node.ping}ms
      </p>
    </Link>
  );
}
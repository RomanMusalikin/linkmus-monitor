import { Activity } from 'lucide-react';
import NodeCard from '../components/cards/NodeCard';
import { useNodes } from '../hooks/useNodes';

export default function Dashboard() {
  // МАГИЯ ЗДЕСЬ: берем данные из нашего Go-бэкенда!
  const { data: nodes, loading, error } = useNodes();

  if (loading && !nodes) {
    return (
      <div className="p-6 flex items-center justify-center h-full text-slate-400">
        <Activity className="w-6 h-6 animate-spin mr-3 text-blue-500" />
        Загрузка узлов инфраструктуры...
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-500/10 border border-red-500/50 text-red-400 p-4 rounded-xl">
          Ошибка связи с мастер-сервером: {error}
        </div>
      </div>
    );
  }

  // Считаем онлайн узлы
  const onlineCount = nodes?.filter(n => n.online)?.length || 0;
  const totalCount = nodes?.length || 0;

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-end">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 mb-1">Главная панель</h1>
          <p className="text-slate-400 text-sm">Обзор состояния узлов инфраструктуры</p>
        </div>
        <div className="text-sm text-slate-400 bg-slate-800 px-3 py-1.5 rounded-lg border border-slate-700">
          Онлайн: <span className="text-green-400 font-medium">{onlineCount}</span> / {totalCount}
        </div>
      </div>
      
      {/* Если БД пустая (агенты не запущены), выводим сообщение */}
      {!nodes || nodes.length === 0 ? (
        <div className="text-center p-10 bg-slate-800/50 rounded-xl border border-dashed border-slate-700 text-slate-500">
          Нет данных от агентов. Ожидание подключений...
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {nodes.map(node => (
            <NodeCard key={node.name} node={node} />
          ))}
        </div>
      )}
    </div>
  );
}
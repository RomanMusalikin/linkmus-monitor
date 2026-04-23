import { Activity, Server, AlertTriangle, Cpu, MemoryStick, HardDrive } from 'lucide-react';
import NodeCard from '../components/cards/NodeCard';
import { useNodesContext } from '../context/NodesContext';
import { useNodeOrder } from '../hooks/useNodeOrder';
import {
  DndContext, closestCenter, PointerSensor, useSensor, useSensors
} from '@dnd-kit/core';
import {
  SortableContext, rectSortingStrategy, useSortable
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

function StatCard({ icon: Icon, label, value, sub, color = 'text-blue-400', bg = 'bg-blue-500/10' }) {
  return (
    <div className="bg-slate-800/80 rounded-2xl p-4 border border-slate-700/50 flex items-center gap-4">
      <div className={`${bg} p-3 rounded-xl flex-shrink-0`}>
        <Icon className={`w-5 h-5 ${color}`} />
      </div>
      <div>
        <div className="text-xs text-slate-500 font-medium mb-0.5">{label}</div>
        <div className={`text-2xl font-bold leading-none ${color}`}>{value}</div>
        {sub && <div className="text-xs text-slate-500 mt-0.5">{sub}</div>}
      </div>
    </div>
  );
}

function SortableCard({ node, onDeleted }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: node.name });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 50 : undefined,
  };
  return (
    <div ref={setNodeRef} style={style}>
      <NodeCard node={node} onDeleted={onDeleted} isDragging={isDragging} dragHandleProps={{ ...attributes, ...listeners }} />
    </div>
  );
}

export default function Dashboard() {
  const { data: nodes, loading, error, refresh } = useNodesContext();
  const { sorted, handleDragEnd } = useNodeOrder(nodes);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }));

  if (loading && !nodes) {
    return (
      <div className="p-6 flex items-center justify-center h-full text-slate-400">
        <Activity className="w-5 h-5 animate-spin mr-2 text-blue-500" />
        Загрузка узлов инфраструктуры...
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-500/10 border border-red-500/30 text-red-400 p-4 rounded-2xl flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 flex-shrink-0" />
          Ошибка связи с мастер-сервером: {error}
        </div>
      </div>
    );
  }

  const online = nodes?.filter(n => n.online) ?? [];
  const offline = nodes?.filter(n => !n.online) ?? [];
  const total = nodes?.length ?? 0;
  const avgCPU = online.length > 0 ? Math.round(online.reduce((s, n) => s + n.cpu, 0) / online.length) : 0;
  const avgRAM = online.length > 0 ? Math.round(online.reduce((s, n) => s + (n.ramTotal > 0 ? (n.ramUsed / n.ramTotal) * 100 : 0), 0) / online.length) : 0;
  const avgDisk = online.length > 0 ? Math.round(online.reduce((s, n) => s + (n.diskUsage || 0), 0) / online.length) : 0;

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-100">Обзор инфраструктуры</h1>
        <p className="text-slate-500 text-sm mt-1">Мониторинг узлов в реальном времени</p>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-5 gap-4 mb-8">
        <StatCard icon={Server} label="Узлы онлайн" value={`${online.length}/${total}`}
          sub={total > 0 ? `${Math.round(online.length/total*100)}% доступно` : '—'}
          color="text-emerald-400" bg="bg-emerald-500/10" />
        <StatCard icon={AlertTriangle} label="Оффлайн" value={offline.length}
          sub={offline.length > 0 ? offline.map(n => n.name).join(', ') : 'Все в сети'}
          color={offline.length > 0 ? 'text-red-400' : 'text-slate-400'}
          bg={offline.length > 0 ? 'bg-red-500/10' : 'bg-slate-700/50'} />
        <StatCard icon={Cpu} label="CPU (среднее)" value={`${avgCPU}%`}
          sub={`по ${online.length} узлам`}
          color={avgCPU > 80 ? 'text-red-400' : avgCPU > 60 ? 'text-amber-400' : 'text-blue-400'}
          bg={avgCPU > 80 ? 'bg-red-500/10' : avgCPU > 60 ? 'bg-amber-500/10' : 'bg-blue-500/10'} />
        <StatCard icon={MemoryStick} label="RAM (среднее)" value={`${avgRAM}%`}
          sub={`по ${online.length} узлам`}
          color={avgRAM > 80 ? 'text-red-400' : avgRAM > 60 ? 'text-amber-400' : 'text-violet-400'}
          bg={avgRAM > 80 ? 'bg-red-500/10' : avgRAM > 60 ? 'bg-amber-500/10' : 'bg-violet-500/10'} />
        <StatCard icon={HardDrive} label="Диск (среднее)" value={`${avgDisk}%`}
          sub={`по ${online.length} узлам`}
          color={avgDisk > 80 ? 'text-red-400' : avgDisk > 60 ? 'text-amber-400' : 'text-cyan-400'}
          bg={avgDisk > 80 ? 'bg-red-500/10' : avgDisk > 60 ? 'bg-amber-500/10' : 'bg-cyan-500/10'} />
      </div>

      <div className="flex items-center gap-3 mb-5">
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Узлы</h2>
        <div className="flex-1 h-px bg-slate-700/50" />
        <span className="text-xs text-slate-500">{total} всего · перетащите для изменения порядка</span>
      </div>

      {!nodes || nodes.length === 0 ? (
        <div className="text-center py-16 bg-slate-800/30 rounded-2xl border border-dashed border-slate-700 text-slate-500">
          <Server className="w-10 h-10 mx-auto mb-3 opacity-30" />
          <p>Нет данных от агентов. Ожидание подключений...</p>
          <p className="text-xs mt-1 text-slate-600">Запустите агент на мониторируемых узлах</p>
        </div>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={sorted.map(n => n.name)} strategy={rectSortingStrategy}>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-5">
              {sorted.map(node => (
                <SortableCard key={node.name} node={node} onDeleted={refresh} />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}
    </div>
  );
}

import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Server, Menu, Pin, PinOff, GripVertical } from 'lucide-react';
import { useNodeOrder } from '../../hooks/useNodeOrder';
import {
  DndContext, closestCenter, PointerSensor, useSensor, useSensors
} from '@dnd-kit/core';
import {
  SortableContext, verticalListSortingStrategy, useSortable
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

function SortableNode({ node, isOpen, isActive, isPinned, onTogglePin, onClick }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: node.name });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <li ref={setNodeRef} style={style} className="group/item relative">
      <Link
        to={`/node/${node.name}`}
        onClick={onClick}
        className={`flex items-center rounded-lg transition-colors overflow-hidden ${
          isActive ? 'bg-slate-700 text-slate-100' : 'text-slate-400 hover:bg-slate-700/50 hover:text-slate-200'
        } ${isOpen ? 'px-3 py-2 gap-3 pr-16' : 'justify-center p-3'}`}
        title={node.displayName || node.name}
      >
        <div className="relative flex items-center justify-center flex-shrink-0">
          <div className={`w-2.5 h-2.5 rounded-full ${node.online ? 'bg-green-500' : 'bg-red-500'}`} />
        </div>
        <span className={`text-sm whitespace-nowrap transition-all duration-300 overflow-hidden flex-1 min-w-0 truncate
          ${isOpen ? 'opacity-100 max-w-[120px]' : 'opacity-0 max-w-0'}`}>
          {node.displayName || node.name}
        </span>
      </Link>

      {isOpen && (
        <div className="absolute right-1 top-1/2 -translate-y-1/2 flex items-center gap-0.5 opacity-0 group-hover/item:opacity-100 transition-opacity">
          <button
            onClick={e => { e.preventDefault(); onTogglePin(node.name); }}
            className={`p-1 rounded hover:bg-slate-600 transition-colors ${isPinned ? 'text-blue-400' : 'text-slate-500'}`}
            title={isPinned ? 'Открепить' : 'Закрепить сверху'}
          >
            {isPinned ? <PinOff className="w-3 h-3" /> : <Pin className="w-3 h-3" />}
          </button>
          <div
            {...attributes} {...listeners}
            className="p-1 rounded hover:bg-slate-600 cursor-grab active:cursor-grabbing text-slate-500"
            title="Перетащить"
          >
            <GripVertical className="w-3 h-3" />
          </div>
        </div>
      )}
    </li>
  );
}

export default function Sidebar({ isOpen, setIsOpen, toggleSidebar, nodes = [] }) {
  const location = useLocation();
  const { sorted, handleDragEnd, togglePin, pinned } = useNodeOrder(nodes);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }));

  const handleLinkClick = () => {
    if (window.innerWidth < 768) setIsOpen(false);
  };

  return (
    <aside className={`
      bg-slate-800 border-r border-slate-700 h-full flex flex-col flex-shrink-0 z-30
      fixed top-0 bottom-0 left-0 transition-all duration-300 ease-in-out
      md:relative md:translate-x-0
      ${isOpen ? 'translate-x-0 w-64' : '-translate-x-full md:translate-x-0 md:w-16 w-64'}
    `}>
      <div className={`h-[73px] border-b border-slate-700/50 flex items-center overflow-hidden flex-shrink-0 transition-all duration-300
        ${isOpen ? 'px-4 gap-2 justify-between' : 'justify-center'}`}>
        <span className={`text-xs font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap transition-all duration-300 overflow-hidden
          ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}>
          Навигация
        </span>
        <button
          onClick={toggleSidebar}
          className="p-2 rounded-lg text-slate-400 hover:bg-slate-700/50 hover:text-slate-200 transition-colors flex-shrink-0"
          title={isOpen ? 'Свернуть меню' : 'Развернуть меню'}
        >
          <Menu className="w-5 h-5" />
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto overflow-x-hidden p-3 space-y-6">
        <div>
          <Link
            to="/"
            onClick={handleLinkClick}
            className={`flex items-center rounded-lg transition-colors overflow-hidden ${
              location.pathname === '/'
                ? 'bg-blue-500/10 text-blue-400'
                : 'text-slate-400 hover:bg-slate-700/50 hover:text-slate-200'
            } ${isOpen ? 'px-3 py-2 gap-3' : 'justify-center p-3'}`}
            title="Дашборд"
          >
            <LayoutDashboard className="w-5 h-5 flex-shrink-0" />
            <span className={`font-medium whitespace-nowrap transition-all duration-300 overflow-hidden
              ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}>
              Дашборд
            </span>
          </Link>
        </div>

        <div>
          <div className={`flex items-center mb-2 overflow-hidden ${isOpen ? 'px-3 gap-2' : 'justify-center'}`}>
            <Server className="w-4 h-4 text-slate-500 flex-shrink-0" />
            <span className={`text-xs font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap transition-all duration-300 overflow-hidden
              ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}>
              Узлы
            </span>
          </div>

          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={sorted.map(n => n.name)} strategy={verticalListSortingStrategy}>
              <ul className="space-y-1">
                {sorted.map(node => (
                  <SortableNode
                    key={node.name}
                    node={node}
                    isOpen={isOpen}
                    isActive={location.pathname === `/node/${node.name}`}
                    isPinned={pinned.includes(node.name)}
                    onTogglePin={togglePin}
                    onClick={handleLinkClick}
                  />
                ))}
              </ul>
            </SortableContext>
          </DndContext>
        </div>
      </nav>

    </aside>
  );
}

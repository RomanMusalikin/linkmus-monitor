import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Server, Menu } from 'lucide-react';

export default function Sidebar({ isOpen, setIsOpen, toggleSidebar, nodes = [] }) {
  const location = useLocation();

  const handleLinkClick = () => {
    if (window.innerWidth < 768) {
      setIsOpen(false);
    }
  };

  return (
    <aside
      className={`
        bg-slate-800 border-r border-slate-700 h-full flex flex-col flex-shrink-0 z-30
        fixed top-0 bottom-0 left-0 transition-all duration-300 ease-in-out
        md:relative md:translate-x-0
        ${isOpen ? 'translate-x-0 w-64' : '-translate-x-full md:translate-x-0 md:w-16 w-64'}
      `}
    >
      {/* Шапка сайдбара */}
      <div className={`h-[73px] border-b border-slate-700/50 flex items-center overflow-hidden flex-shrink-0 transition-all duration-300
        ${isOpen ? 'px-4 gap-2 justify-between' : 'justify-center'}`}>
        <span
          className={`text-xs font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap transition-all duration-300 overflow-hidden
            ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}
        >
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

        {/* Главная */}
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
            <span
              className={`font-medium whitespace-nowrap transition-all duration-300 overflow-hidden
                ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}
            >
              Дашборд
            </span>
          </Link>
        </div>

        {/* Узлы */}
        <div>
          <div className={`flex items-center mb-2 overflow-hidden ${isOpen ? 'px-3 gap-2' : 'justify-center'}`}>
            <Server className="w-4 h-4 text-slate-500 flex-shrink-0" />
            <span
              className={`text-xs font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap transition-all duration-300 overflow-hidden
                ${isOpen ? 'opacity-100 max-w-[200px]' : 'opacity-0 max-w-0'}`}
            >
              Узлы
            </span>
          </div>

          <ul className="space-y-1">
            {nodes.map(node => {
              const isActive = location.pathname === `/node/${node.name}`;
              return (
                <li key={node.name}>
                  <Link
                    to={`/node/${node.name}`}
                    onClick={handleLinkClick}
                    className={`flex items-center rounded-lg transition-colors overflow-hidden ${
                      isActive
                        ? 'bg-slate-700 text-slate-100'
                        : 'text-slate-400 hover:bg-slate-700/50 hover:text-slate-200'
                    } ${isOpen ? 'px-3 py-2 gap-3' : 'justify-center p-3'}`}
                    title={node.name}
                  >
                    {/* Точка статуса */}
                    <div className="relative flex items-center justify-center flex-shrink-0">
                      <div className={`w-2.5 h-2.5 rounded-full ${node.online ? 'bg-green-500' : 'bg-red-500'}`} />
                      {!isOpen && isActive && (
                        <div className={`absolute inset-0 rounded-full border-2 animate-pulse ${node.online ? 'border-green-500/50' : 'border-red-500/50'} -m-[2px] w-[14px] h-[14px]`} />
                      )}
                    </div>

                    <span
                      className={`text-sm whitespace-nowrap transition-all duration-300 overflow-hidden
                        ${isOpen ? 'opacity-100 max-w-[160px]' : 'opacity-0 max-w-0'}`}
                    >
                      {node.name}
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      </nav>

      {/* Подвал */}
      <div className="p-4 border-t border-slate-700/50 text-xs text-slate-500 text-center flex-shrink-0 overflow-hidden">
        <span className={`transition-all duration-300 ${isOpen ? 'opacity-100' : 'opacity-0 md:opacity-100'}`}>
          {isOpen ? 'LinkMus v1.0' : 'v1.0'}
        </span>
      </div>
    </aside>
  );
}

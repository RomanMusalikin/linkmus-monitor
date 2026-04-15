import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Server, Menu } from 'lucide-react';

export default function Sidebar({ isOpen, setIsOpen, toggleSidebar, nodes = [] }) {
  const location = useLocation();

  // Функция для закрытия меню на мобилках после клика по ссылке
  const handleLinkClick = () => {
    if (window.innerWidth < 768) { // 768px - брейкпоинт md в Tailwind
      setIsOpen(false);
    }
  };

  return (
    <aside 
      className={`
        bg-slate-800 border-r border-slate-700 h-full flex flex-col flex-shrink-0 transition-all duration-300 ease-in-out z-30
        /* Мобильные стили: абсолютное позиционирование, уезжает за экран */
        fixed top-0 bottom-0 left-0 ${isOpen ? 'translate-x-0' : '-translate-x-full'}
        /* ПК стили: относительное позиционирование, меняет ширину (w-64 -> w-20) */
        md:relative md:translate-x-0 ${isOpen ? 'w-64' : 'w-20'}
      `}
    >
      {/* Шапка сайдбара — по высоте совпадает с Header */}
      <div className="h-[73px] border-b border-slate-700/50 flex items-center px-4 justify-between">
        {isOpen && (
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Навигация</span>
        )}
        <button
          onClick={toggleSidebar}
          className={`p-2 rounded-lg text-slate-400 hover:bg-slate-700/50 hover:text-slate-200 transition-colors flex-shrink-0 ${!isOpen ? 'mx-auto' : ''}`}
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
            className={`flex items-center rounded-lg transition-colors ${
              location.pathname === '/' 
                ? 'bg-blue-500/10 text-blue-400' 
                : 'text-slate-400 hover:bg-slate-700/50 hover:text-slate-200'
            } ${isOpen ? 'px-3 py-2 gap-3' : 'justify-center p-3'}`}
            title="Дашборд"
          >
            <LayoutDashboard className="w-5 h-5 flex-shrink-0" />
            {isOpen && <span className="font-medium whitespace-nowrap">Дашборд</span>}
          </Link>
        </div>

        {/* Узлы */}
        <div>
          <div className={`flex items-center mb-2 ${isOpen ? 'px-3 gap-2' : 'justify-center'}`}>
            <Server className="w-4 h-4 text-slate-500" />
            {isOpen && <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Узлы</span>}
          </div>
          
          <ul className="space-y-1">
            {nodes.map(node => {
              const isActive = location.pathname === `/node/${node.name}`;
              return (
                <li key={node.name}>
                  <Link
                    to={`/node/${node.name}`}
                    onClick={handleLinkClick}
                    className={`flex items-center rounded-lg transition-colors ${
                      isActive 
                        ? 'bg-slate-700 text-slate-100' 
                        : 'text-slate-400 hover:bg-slate-700/50 hover:text-slate-200'
                    } ${isOpen ? 'px-3 py-2 gap-3' : 'justify-center p-3'}`}
                    title={node.name} // Показываем имя при наведении на свернутом меню
                  >
                    {/* Точка статуса */}
                    <div className="relative flex items-center justify-center">
                      <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${node.online ? 'bg-green-500' : 'bg-red-500'}`} />
                      {/* Если меню свернуто и узел активен, добавляем колечко вокруг точки для понятности */}
                      {!isOpen && isActive && (
                        <div className={`absolute inset-0 w-full h-full rounded-full border-2 animate-pulse ${node.online ? 'border-green-500/50' : 'border-red-500/50'} -m-[2px] w-[14px] h-[14px]`} />
                      )}
                    </div>
                    
                    {isOpen && <span className="text-sm truncate whitespace-nowrap">{node.name}</span>}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      </nav>

      {/* Подвал */}
      <div className={`p-4 border-t border-slate-700/50 text-xs text-slate-500 ${isOpen ? 'text-center' : 'text-center text-[10px] hidden md:block'}`}>
        {isOpen ? 'LinkMus v1.0' : 'v1.0'}
      </div>
    </aside>
  );
}
import { useState, useEffect } from 'react';
import { Activity, LogOut } from 'lucide-react';
import { logout } from '../../lib/api';

export default function Header({ onlineCount = 0, totalCount = 0, onLogout }) {
  const [time, setTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  async function handleLogout() {
    await logout();
    onLogout?.();
  }

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-4 sm:px-6 py-4 flex items-center justify-between z-10 relative">
      <div className="flex items-center gap-3 sm:gap-4">
        <div className="hidden sm:flex bg-blue-500/20 p-2 rounded-lg">
          <Activity className="text-blue-500 w-5 h-5 sm:w-6 sm:h-6" />
        </div>
        <h1 className="text-lg sm:text-xl font-bold tracking-tight text-slate-100">LinkMus Monitor</h1>
      </div>

      <div className="flex items-center gap-3 sm:gap-4">
        <div className="text-slate-300 font-mono text-xs sm:text-sm bg-slate-900/50 px-2 sm:px-3 py-1 sm:py-1.5 rounded-md border border-slate-700/50">
          {time.toLocaleTimeString()}
        </div>
        <div className="text-slate-400 text-sm hidden md:block">
          Онлайн: <span className="text-green-400 font-medium">{onlineCount}</span> / {totalCount}
        </div>
        <button
          onClick={handleLogout}
          title="Выйти"
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 border border-transparent hover:border-red-500/20 transition-all duration-200 text-sm"
        >
          <LogOut className="w-4 h-4" />
          <span className="hidden sm:inline">Выйти</span>
        </button>
      </div>
    </header>
  );
}

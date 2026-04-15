import { useState, useEffect } from 'react';
import { Activity } from 'lucide-react';

export default function Header({ onlineCount = 0, totalCount = 0 }) {
  const [time, setTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-4 sm:px-6 py-4 flex items-center justify-between z-10 relative">
      <div className="flex items-center gap-3 sm:gap-4">
        {/* Логотип */}
        <div className="hidden sm:flex bg-blue-500/20 p-2 rounded-lg">
          <Activity className="text-blue-500 w-5 h-5 sm:w-6 sm:h-6" />
        </div>
        <h1 className="text-lg sm:text-xl font-bold tracking-tight text-slate-100">LinkMus Monitor</h1>
      </div>
      
      <div className="flex items-center gap-4 sm:gap-6">
        <div className="text-slate-300 font-mono text-xs sm:text-sm bg-slate-900/50 px-2 sm:px-3 py-1 sm:py-1.5 rounded-md border border-slate-700/50">
          {time.toLocaleTimeString()}
        </div>
        <div className="text-slate-400 text-sm hidden md:block">
          Онлайн: <span className="text-green-400 font-medium">{onlineCount}</span> / {totalCount}
        </div>
      </div>
    </header>
  );
}
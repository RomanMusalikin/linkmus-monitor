import { useState, useEffect, useRef } from 'react';
import { Activity, LogOut, UserPlus, X, Settings, Menu } from 'lucide-react';
import { Link } from 'react-router-dom';
import { logout, createUser } from '../../lib/api';

function CreateUserModal({ onClose }) {
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef(null);

  useEffect(() => { inputRef.current?.focus(); }, []);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    if (!login.trim() || !password) { setError('Заполните все поля'); return; }
    if (password !== password2) { setError('Пароли не совпадают'); return; }
    setLoading(true);
    try {
      await createUser(login.trim(), password);
      setSuccess(true);
      setTimeout(onClose, 1200);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6 w-full max-w-sm shadow-2xl">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-base font-semibold text-slate-100">Новый пользователь</h2>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {success ? (
          <div className="text-center py-4 text-emerald-400 font-medium">Пользователь создан ✓</div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs text-slate-500 mb-1.5 uppercase tracking-wider">Логин</label>
              <input ref={inputRef} type="text" value={login} onChange={e => setLogin(e.target.value)}
                autoComplete="off" placeholder="username"
                className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-600 text-slate-200 text-sm
                  focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-all" />
            </div>
            <div>
              <label className="block text-xs text-slate-500 mb-1.5 uppercase tracking-wider">Пароль</label>
              <input type="password" value={password} onChange={e => setPassword(e.target.value)}
                autoComplete="new-password" placeholder="••••••••"
                className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-600 text-slate-200 text-sm
                  focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-all" />
            </div>
            <div>
              <label className="block text-xs text-slate-500 mb-1.5 uppercase tracking-wider">Повторите пароль</label>
              <input type="password" value={password2} onChange={e => setPassword2(e.target.value)}
                autoComplete="new-password" placeholder="••••••••"
                className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-600 text-slate-200 text-sm
                  focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-all" />
            </div>
            {error && (
              <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">{error}</div>
            )}
            <button type="submit" disabled={loading}
              className="w-full py-2.5 rounded-lg bg-blue-500/20 text-blue-400 border border-blue-500/30
                hover:bg-blue-500/30 hover:text-blue-300 transition-all font-medium text-sm disabled:opacity-50">
              {loading ? '...' : 'Создать'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}

export default function Header({ onlineCount = 0, totalCount = 0, onLogout, version, onMenuClick }) {
  const [time, setTime] = useState(new Date());
  const [showCreateUser, setShowCreateUser] = useState(false);

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  async function handleLogout() {
    await logout();
    onLogout?.();
  }

  return (
    <>
      <header className="bg-slate-800 border-b border-slate-700 px-4 sm:px-6 py-4 flex items-center justify-between z-10 relative">
        <div className="flex items-center gap-3 sm:gap-4">
          <button
            onClick={onMenuClick}
            className="md:hidden p-2 rounded-lg text-slate-400 hover:bg-slate-700/50 hover:text-slate-200 transition-colors"
            title="Меню"
          >
            <Menu className="w-5 h-5" />
          </button>
          <div className="hidden sm:flex bg-blue-500/20 p-2 rounded-lg">
            <Activity className="text-blue-500 w-5 h-5 sm:w-6 sm:h-6" />
          </div>
          <Link to="/" className="hover:opacity-80 transition-opacity">
            <h1 className="text-lg sm:text-xl font-bold tracking-tight text-slate-100">LinkMus Monitor</h1>
            {version && <span className="text-xs text-slate-500">{version}</span>}
          </Link>
        </div>

        <div className="flex items-center gap-3 sm:gap-4">
          <div className="text-slate-300 font-mono text-xs sm:text-sm bg-slate-900/50 px-2 sm:px-3 py-1 sm:py-1.5 rounded-md border border-slate-700/50">
            {time.toLocaleTimeString()}
          </div>
          <div className="text-slate-400 text-sm hidden md:block">
            Онлайн: <span className="text-green-400 font-medium">{onlineCount}</span> / {totalCount}
          </div>
          <button
            onClick={() => setShowCreateUser(true)}
            title="Создать пользователя"
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-slate-400 hover:text-blue-400 hover:bg-blue-500/10 border border-transparent hover:border-blue-500/20 transition-all duration-200 text-sm"
          >
            <UserPlus className="w-4 h-4" />
            <span className="hidden sm:inline">Пользователи</span>
          </button>
          <Link to="/settings"
            title="Настройки"
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-700/50 border border-transparent hover:border-slate-600/30 transition-all duration-200 text-sm"
          >
            <Settings className="w-4 h-4" />
            <span className="hidden sm:inline">Настройки</span>
          </Link>
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
      {showCreateUser && <CreateUserModal onClose={() => setShowCreateUser(false)} />}
    </>
  );
}

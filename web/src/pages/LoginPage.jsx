import { useState, useRef, useCallback } from 'react';
import { login, register } from '../lib/api';

function LampSVG({ lit, flickering, pulling, onClick }) {
  // Цвет свечения — синий как акцент панели мониторинга
  const glowColor = '#38bdf8'; // sky-400
  const litFill   = '#e0f4ff';
  const litStroke = '#38bdf8';
  const litBase   = '#1e6fa8';
  const litBase2  = '#155e8a';
  const wireColor = lit ? '#1e6fa8' : '#5a5a6a';

  return (
    <svg
      width="120"
      height="290"
      viewBox="0 0 120 290"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      onClick={onClick}
      className={flickering ? 'lamp-flickering' : ''}
      style={{ cursor: 'pointer', display: 'block', overflow: 'visible' }}
    >
      {/* Провод от потолка */}
      <line x1="60" y1="0" x2="60" y2="40" stroke="#4a4a5a" strokeWidth="3" strokeLinecap="round"/>

      {/* Патрон */}
      <rect x="46" y="40" width="28" height="18" rx="4" fill="#3a3a4a"/>
      <rect x="49" y="43" width="22" height="12" rx="3" fill="#2a2a38"/>

      {/* Свечение */}
      <ellipse cx="60" cy="97" rx="56" ry="56" fill="url(#glowGrad)"
        style={{ opacity: lit ? 0.5 : 0, transition: 'opacity 0.9s ease' }}/>

      {/* Колба */}
      <ellipse cx="60" cy="97" rx="32" ry="36"
        fill={lit ? litFill : '#1e1e2e'}
        stroke={lit ? litStroke : '#3a3a4a'}
        strokeWidth="2"
        style={{ transition: 'fill 0.7s ease, stroke 0.7s ease' }}
      />

      {/* Нить — верхняя */}
      <path d="M52 90 Q56 84 60 90 Q64 96 68 90"
        stroke={lit ? '#0ea5e9' : '#3a3a58'}
        strokeWidth="2" fill="none" strokeLinecap="round"
        style={{ transition: 'stroke 0.6s ease' }}
      />
      {/* Нить — нижняя */}
      <path d="M54 96 Q58 90 62 96 Q66 102 70 96"
        stroke={lit ? '#38bdf8' : '#2e2e48'}
        strokeWidth="2" fill="none" strokeLinecap="round"
        style={{ transition: 'stroke 0.6s ease' }}
      />

      {/* Цоколь */}
      <rect x="50" y="130" width="20" height="8" rx="2"
        fill={lit ? litBase : '#2a2a38'}
        style={{ transition: 'fill 0.7s ease' }}/>
      <rect x="48" y="138" width="24" height="6" rx="2"
        fill={lit ? litBase2 : '#232330'}
        style={{ transition: 'fill 0.7s ease' }}/>

      {/* Шнурок — всегда виден */}
      <g className={pulling ? 'cord-pulling' : ''}>
        <line
          x1="60" y1="144" x2="60" y2="232"
          stroke={wireColor}
          strokeWidth="3" strokeLinecap="round"
          style={{ transition: 'stroke 0.8s ease' }}
        />
        <circle cx="60" cy="240" r="8" fill="url(#knobGrad)"/>
      </g>

      <defs>
        <radialGradient id="glowGrad" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor={glowColor} stopOpacity="0.85"/>
          <stop offset="100%" stopColor={glowColor} stopOpacity="0"/>
        </radialGradient>
        <radialGradient id="knobGrad" cx="40%" cy="35%" r="60%">
          <stop offset="0%" stopColor="#9a9aaa"/>
          <stop offset="100%" stopColor="#3a3a4a"/>
        </radialGradient>
      </defs>
    </svg>
  );
}

export default function LoginPage({ mode, onAuth }) {
  const isSetup = mode === 'setup';

  const [lit, setLit]           = useState(false);
  const [flickering, setFlickering] = useState(false);
  const [pulling, setPulling]   = useState(false);
  const [formVisible, setFormVisible] = useState(false);
  const [focusedField, setFocusedField] = useState(null);

  const [loginVal, setLoginVal] = useState('');
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [error, setError]       = useState('');
  const [loading, setLoading]   = useState(false);

  const pullingRef = useRef(false);

  const handleLampClick = useCallback(() => {
    if (pullingRef.current || flickering) return;

    if (lit) {
      // Выключение — дёрнуть шнурок, потом мигание, потом темно
      pullingRef.current = true;
      setPulling(true);
      setTimeout(() => {
        setPulling(false);
        setFormVisible(false);
        setFlickering(true);
        setTimeout(() => {
          setFlickering(false);
          setLit(false);
          pullingRef.current = false;
        }, 700);
      }, 700);
      return;
    }

    // Включение — дёрнуть шнурок
    pullingRef.current = true;
    setPulling(true);
    setTimeout(() => {
      setPulling(false);
      setLit(true);
      setTimeout(() => setFormVisible(true), 400);
      pullingRef.current = false;
    }, 700);
  }, [lit, flickering]);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    if (isSetup && password !== password2) { setError('Пароли не совпадают'); return; }
    if (!loginVal.trim() || !password)     { setError('Заполните все поля');   return; }
    setLoading(true);
    try {
      if (isSetup) await register(loginVal.trim(), password);
      else         await login(loginVal.trim(), password);
      onAuth();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  function fieldStyle(name) {
    const focused = focusedField === name;
    return {
      ...inputBase,
      border: focused ? '1px solid rgba(56,189,248,0.6)' : '1px solid rgba(100,116,139,0.3)',
      boxShadow: focused ? '0 0 0 3px rgba(56,189,248,0.09)' : 'none',
    };
  }
  const fp = name => ({
    onFocus: () => setFocusedField(name),
    onBlur:  () => setFocusedField(null),
  });

  const bgGlow = lit
    ? 'radial-gradient(ellipse 70% 55% at 50% 0%, rgba(56,189,248,0.18) 0%, #0f1117 55%)'
    : '#0f1117';

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      background: bgGlow,
      transition: 'background 1.4s ease',
      overflow: 'hidden',
    }}>
      {/* Потолочная полоса */}
      <div style={{
        width: '100%', height: 6, flexShrink: 0,
        background: lit ? 'rgba(56,189,248,0.15)' : '#111118',
        transition: 'background 1.2s ease',
      }}/>

      {/* Лампа */}
      <div className={lit && !flickering ? 'lamp-lit' : ''} style={{ marginTop: 0, transition: 'filter 0.8s ease' }}>
        <LampSVG lit={lit} flickering={flickering} pulling={pulling} onClick={handleLampClick} />
      </div>

      {/* Форма */}
      <div style={{
        width: '100%', maxWidth: 420, padding: '0 24px',
        marginTop: formVisible ? 4 : 0,
        opacity: formVisible ? 1 : 0,
        transform: formVisible ? 'translateY(0)' : 'translateY(28px)',
        transition: 'opacity 0.55s cubic-bezier(0.22,1,0.36,1), transform 0.55s cubic-bezier(0.22,1,0.36,1)',
        pointerEvents: formVisible ? 'auto' : 'none',
      }}>
        {/* Заголовок */}
        <div style={{ textAlign: 'center', marginBottom: 22 }}>
          <div style={{ fontSize: 11, letterSpacing: '0.2em', color: '#38bdf8', textTransform: 'uppercase', marginBottom: 8, opacity: 0.7 }}>
            LinkMus Monitor
          </div>
          <h1 style={{ fontSize: 22, fontWeight: 700, color: '#f1f5f9', margin: 0 }}>
            {isSetup ? 'Создайте учётную запись' : 'Добро пожаловать'}
          </h1>
          {isSetup && (
            <p style={{ color: '#64748b', fontSize: 13, marginTop: 8 }}>
              Первый запуск — настройте администратора
            </p>
          )}
        </div>

        {/* Карточка */}
        <form onSubmit={handleSubmit} style={{
          background: 'rgba(15,23,42,0.85)',
          border: '1px solid rgba(56,189,248,0.1)',
          borderRadius: 16,
          padding: '26px 26px',
          backdropFilter: 'blur(14px)',
          boxShadow: '0 8px 48px rgba(0,0,0,0.7), 0 0 0 1px rgba(56,189,248,0.05)',
        }}>
          <div style={{ marginBottom: 16 }}>
            <label style={labelStyle}>ЛОГИН</label>
            <input type="text" value={loginVal} onChange={e => setLoginVal(e.target.value)}
              autoComplete="username" placeholder="admin"
              style={fieldStyle('login')} {...fp('login')}/>
          </div>

          <div style={{ marginBottom: isSetup ? 16 : 22 }}>
            <label style={labelStyle}>ПАРОЛЬ</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)}
              autoComplete={isSetup ? 'new-password' : 'current-password'} placeholder="••••••••"
              style={fieldStyle('password')} {...fp('password')}/>
          </div>

          {isSetup && (
            <div style={{ marginBottom: 22 }}>
              <label style={labelStyle}>ПОВТОРИТЕ ПАРОЛЬ</label>
              <input type="password" value={password2} onChange={e => setPassword2(e.target.value)}
                autoComplete="new-password" placeholder="••••••••"
                style={fieldStyle('password2')} {...fp('password2')}/>
            </div>
          )}

          {error && (
            <div style={{
              background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.2)',
              borderRadius: 8, padding: '10px 14px', color: '#f87171',
              fontSize: 13, marginBottom: 18,
            }}>
              {error}
            </div>
          )}

          <button type="submit" disabled={loading} style={{
            width: '100%', padding: '12px 0', borderRadius: 10, border: 'none',
            background: loading
              ? 'rgba(56,189,248,0.1)'
              : 'linear-gradient(135deg, #0284c7 0%, #38bdf8 50%, #0284c7 100%)',
            color: loading ? '#475569' : '#0f172a',
            fontWeight: 700, fontSize: 15,
            cursor: loading ? 'not-allowed' : 'pointer',
            letterSpacing: '0.04em', transition: 'all 0.25s',
            boxShadow: loading ? 'none' : '0 4px 20px rgba(56,189,248,0.2)',
          }}>
            {loading ? '...' : isSetup ? 'Создать и войти' : 'Войти'}
          </button>
        </form>
      </div>

      {lit && <DustParticles />}
    </div>
  );
}

const labelStyle = {
  display: 'block', fontSize: 12, color: '#64748b',
  marginBottom: 6, letterSpacing: '0.06em',
};

const inputBase = {
  width: '100%', padding: '11px 14px', borderRadius: 8,
  background: 'rgba(30,41,59,0.7)', color: '#e2e8f0',
  fontSize: 14, outline: 'none', boxSizing: 'border-box',
  transition: 'border-color 0.2s, box-shadow 0.2s',
};

function DustParticles() {
  const particles = Array.from({ length: 18 }, (_, i) => ({
    id: i,
    left: `${10 + (i * 4.5 + 7) % 80}%`,
    animDuration: `${5 + (i * 1.3) % 8}s`,
    animDelay: `${(i * 0.7) % 4}s`,
    size: 1 + (i % 3) * 0.8,
    opacity: 0.06 + (i % 4) * 0.035,
  }));
  return (
    <div style={{ position: 'fixed', inset: 0, pointerEvents: 'none', zIndex: 0 }}>
      {particles.map(p => (
        <div key={p.id} className="dust-particle" style={{
          position: 'absolute', left: p.left, bottom: '-10px',
          width: p.size, height: p.size, borderRadius: '50%',
          background: '#38bdf8', opacity: p.opacity,
          animationDuration: p.animDuration, animationDelay: p.animDelay,
        }}/>
      ))}
    </div>
  );
}

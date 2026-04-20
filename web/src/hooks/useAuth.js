import { useState, useEffect } from 'react';
import { checkNeedSetup } from '../lib/api';

// Возвращает: { status: 'loading'|'setup'|'login'|'ok', refresh }
export function useAuth() {
  const [status, setStatus] = useState('loading');

  async function check() {
    const token = localStorage.getItem('mon_token');
    if (token) {
      // Быстро проверяем токен запросом к защищённому ресурсу
      try {
        const res = await fetch('/api/nodes', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (res.ok) {
          setStatus('ok');
          return;
        }
      } catch (_) {}
      localStorage.removeItem('mon_token');
    }
    // Нет токена — определяем нужна ли регистрация
    try {
      const needSetup = await checkNeedSetup();
      setStatus(needSetup ? 'setup' : 'login');
    } catch (_) {
      setStatus('login');
    }
  }

  useEffect(() => { check(); }, []);

  return { status, refresh: check };
}

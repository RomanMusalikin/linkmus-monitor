import { useCallback } from 'react';
import { fetchNodes } from '../lib/api';
import useAutoRefresh from './useAutoRefresh';

export function useNodes() {
  // Оборачиваем вызов API в useCallback, чтобы избежать лишних ререндеров
  const fetcher = useCallback(() => fetchNodes(), []);
  
  // Опрашиваем сервер каждые 5 секунд (5000 мс)
  return useAutoRefresh(fetcher, 5000); 
}
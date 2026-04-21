import { useCallback } from 'react';
import { fetchNodes } from '../lib/api';
import useAutoRefresh from './useAutoRefresh';

export function useNodes() {
  const fetcher = useCallback(() => fetchNodes(), []);
  return useAutoRefresh(fetcher, 5000);
}

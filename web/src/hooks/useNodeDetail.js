import { useCallback } from 'react';
import { fetchNodeDetail } from '../lib/api';
import useAutoRefresh from './useAutoRefresh';

export function useNodeDetail(nodeId) {
  const fetcher = useCallback(() => fetchNodeDetail(nodeId), [nodeId]);
  
  // Детали тоже опрашиваем раз в 5 секунд
  return useAutoRefresh(fetcher, 5000); 
}
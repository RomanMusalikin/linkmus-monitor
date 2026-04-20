import { useState, useEffect, useCallback } from 'react';

export default function useAutoRefresh(fetchFn, intervalMs = 5000) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const loadData = useCallback(async () => {
    try {
      const result = await fetchFn();
      setData(result);
      setError(null); // Сбрасываем ошибку при успешном запросе
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [fetchFn]);

  useEffect(() => {
    let isMounted = true;
    let timerId;

    // Рекурсивная функция для безопасного опроса
    const tick = async () => {
      if (isMounted) await loadData();
      if (isMounted) timerId = setTimeout(tick, intervalMs);
    };

    tick();

    // Очистка при размонтировании компонента
    return () => {
      isMounted = false;
      clearTimeout(timerId);
    };
  }, [loadData, intervalMs]);

  return { data, loading, error, refresh: loadData };
}
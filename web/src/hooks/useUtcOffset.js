import { useState, useEffect } from 'react';
import { getAlertSettings } from '../lib/api';

let cachedOffset = null;
const listeners = new Set();

export function getUtcOffset() {
  if (cachedOffset !== null) return cachedOffset;
  const stored = localStorage.getItem('utcOffset');
  return stored !== null ? parseInt(stored, 10) : 0;
}

export function setUtcOffsetGlobal(offset) {
  cachedOffset = offset;
  localStorage.setItem('utcOffset', String(offset));
  listeners.forEach(fn => fn(offset));
}

export function useUtcOffset() {
  const [offset, setOffset] = useState(getUtcOffset);

  useEffect(() => {
    const handler = (v) => setOffset(v);
    listeners.add(handler);
    return () => listeners.delete(handler);
  }, []);

  // При первом монтировании синхронизируем с сервером
  useEffect(() => {
    getAlertSettings()
      .then(s => {
        if (s.utcOffset !== undefined) {
          setUtcOffsetGlobal(s.utcOffset);
        }
      })
      .catch(() => {});
  }, []);

  return offset;
}

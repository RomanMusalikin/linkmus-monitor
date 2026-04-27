import { useState, useEffect } from 'react';

export function useVersion() {
  const [version, setVersion] = useState(null);

  useEffect(() => {
    fetch('/api/version')
      .then(r => r.json())
      .then(d => setVersion(d.version))
      .catch(() => {});
  }, []);

  return version;
}

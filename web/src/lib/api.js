const API_BASE = '/api';

function getToken() {
  return localStorage.getItem('mon_token') || '';
}

function authHeaders() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${getToken()}`,
  };
}

export async function fetchNodes(full) {
  const url = full ? `${API_BASE}/nodes?full=true` : `${API_BASE}/nodes`;
  const res = await fetch(url, { headers: authHeaders() });
  if (res.status === 401) throw Object.assign(new Error('unauthorized'), { status: 401 });
  if (!res.ok) throw new Error('Ошибка получения списка узлов');
  return res.json();
}

export async function fetchNodeDetail(nodeId) {
  const res = await fetch(`${API_BASE}/nodes/${nodeId}`, { headers: authHeaders() });
  if (res.status === 401) throw Object.assign(new Error('unauthorized'), { status: 401 });
  if (!res.ok) throw new Error(`Ошибка получения данных для узла ${nodeId}`);
  return res.json();
}

export async function checkNeedSetup() {
  const res = await fetch(`${API_BASE}/auth/setup`);
  const data = await res.json();
  return data.needSetup;
}

export async function register(login, password, email = '') {
  const res = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login, password, email }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка регистрации');
  }
  const data = await res.json();
  localStorage.setItem('mon_token', data.token);
}

export async function login(loginVal, password) {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: loginVal, password }),
  });
  if (!res.ok) throw new Error('Неверный логин или пароль');
  const data = await res.json();
  localStorage.setItem('mon_token', data.token);
}

export async function fetchNodeHistory(name, range) {
  const res = await fetch(`${API_BASE}/history/${encodeURIComponent(name)}?range=${range}`, {
    headers: authHeaders(),
  });
  if (res.status === 401) throw Object.assign(new Error('unauthorized'), { status: 401 });
  if (!res.ok) throw new Error('Ошибка получения истории');
  return res.json();
}

export async function renameNode(name, alias) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}/alias`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify({ alias }),
  });
  if (!res.ok) throw new Error('Ошибка переименования узла');
}

export async function deleteNode(name) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error('Ошибка удаления узла');
}

export async function createUser(login, password) {
  const res = await fetch(`${API_BASE}/auth/users`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ login, password }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка создания пользователя');
  }
}

export async function logout() {
  await fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
    headers: authHeaders(),
  });
  localStorage.removeItem('mon_token');
}

export async function getAlertSettings() {
  const res = await fetch(`${API_BASE}/settings/alerts`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения настроек');
  return res.json();
}

export async function saveAlertSettings(settings) {
  const res = await fetch(`${API_BASE}/settings/alerts`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(settings),
  });
  if (!res.ok) throw new Error('Ошибка сохранения настроек');
}

export async function sendTestEmail() {
  const res = await fetch(`${API_BASE}/settings/alerts`, {
    method: 'POST',
    headers: authHeaders(),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка отправки');
  }
}

export async function getPortSettings() {
  const res = await fetch(`${API_BASE}/settings/ports`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения настроек портов');
  return res.json();
}

export async function savePortSettings(settings) {
  const res = await fetch(`${API_BASE}/settings/ports`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(settings),
  });
  if (!res.ok) throw new Error('Ошибка сохранения настроек портов');
}

export async function getNodePortOverride(name) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}/ports`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения настроек портов узла');
  return res.json();
}

export async function saveNodePortOverride(name, ports) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}/ports`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(ports),
  });
  if (!res.ok) throw new Error('Ошибка сохранения настроек портов узла');
}

export async function getCustomServices() {
  const res = await fetch(`${API_BASE}/settings/services`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения списка сервисов');
  return res.json();
}

export async function createCustomService(name, port) {
  const res = await fetch(`${API_BASE}/settings/services`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ name, port }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка создания сервиса');
  }
  return res.json();
}

export async function deleteCustomService(id) {
  const res = await fetch(`${API_BASE}/settings/services/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error('Ошибка удаления сервиса');
}

export async function getNodeServiceVisibility(name) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}/visibility`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения настроек видимости');
  return res.json();
}

export async function saveNodeServiceVisibility(name, visibility) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}/visibility`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(visibility),
  });
  if (!res.ok) throw new Error('Ошибка сохранения настроек видимости');
}

export async function sendTestTelegram() {
  const res = await fetch(`${API_BASE}/settings/alerts/test-telegram`, {
    method: 'POST',
    headers: authHeaders(),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка отправки');
  }
}

export async function getGigachatSettings() {
  const res = await fetch(`${API_BASE}/settings/gigachat`, { headers: authHeaders() });
  if (!res.ok) throw new Error('Ошибка получения настроек GigaChat');
  return res.json();
}

export async function saveGigachatSettings(settings) {
  const res = await fetch(`${API_BASE}/settings/gigachat`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(settings),
  });
  if (!res.ok) throw new Error('Ошибка сохранения настроек GigaChat');
}

export async function generateReport(nodes, period, from, to) {
  const body = { nodes, period };
  if (from && to) { body.from = from; body.to = to; }
  const res = await fetch(`${API_BASE}/report`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Ошибка генерации отчёта');
  }
  return res.json();
}

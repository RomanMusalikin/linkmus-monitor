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

export async function fetchNodes() {
  const res = await fetch(`${API_BASE}/nodes`, { headers: authHeaders() });
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

export async function register(login, password) {
  const res = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login, password }),
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

export async function deleteNode(name) {
  const res = await fetch(`${API_BASE}/nodes/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });
  if (!res.ok) throw new Error('Ошибка удаления узла');
}

export async function logout() {
  await fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
    headers: authHeaders(),
  });
  localStorage.removeItem('mon_token');
}

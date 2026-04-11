// src/lib/api.js
const API_BASE = '/api';

export async function fetchNodes() {
  const response = await fetch(`${API_BASE}/nodes`);
  if (!response.ok) throw new Error('Ошибка получения списка узлов');
  return response.json();
}

export async function fetchNodeDetail(nodeId) {
  const response = await fetch(`${API_BASE}/nodes/${nodeId}`);
  if (!response.ok) throw new Error(`Ошибка получения данных для узла ${nodeId}`);
  return response.json();
}
// Сдвигает строку времени "HH:MM" или "HH:MM:SS" на offsetHours часов.
// Используется для перевода UTC-меток с сервера в локальное время.
export function shiftTime(timeStr, offsetHours) {
  if (!timeStr || offsetHours === 0) return timeStr;
  const parts = timeStr.split(':').map(Number);
  const totalMin = parts[0] * 60 + parts[1] + offsetHours * 60;
  const h = (((totalMin / 60) % 24) + 24) % 24;
  const m = ((totalMin % 60) + 60) % 60;
  const hh = String(Math.floor(h)).padStart(2, '0');
  const mm = String(m).padStart(2, '0');
  if (parts.length === 3) {
    return `${hh}:${mm}:${String(parts[2]).padStart(2, '0')}`;
  }
  return `${hh}:${mm}`;
}

// Возвращает текущее время (HH:MM:SS) с учётом UTC-смещения,
// независимо от часового пояса браузера.
export function nowWithOffset(offsetHours) {
  const now = new Date();
  const utcMs = now.getTime() + now.getTimezoneOffset() * 60000;
  const local = new Date(utcMs + offsetHours * 3600000);
  return local.toTimeString().slice(0, 8);
}

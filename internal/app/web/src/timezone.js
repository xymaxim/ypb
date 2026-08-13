const pad = (n, len = 2) => String(n).padStart(len, '0');

export function parseTzOffset(raw) {
  const m = /^([+-])(\d{1,2})(?::([0-5]\d))?$/.exec(raw);
  if (!m) return null;
  const h = parseInt(m[2], 10);
  if (h > 23) return null;
  const min = m[3] ? parseInt(m[3], 10) : 0;
  const ms = (h * 3600 + min * 60) * 1000;
  return { offsetMs: m[1] === '-' ? -ms : ms, offsetLabel: `${m[1]}${pad(h)}:${pad(min)}` };
}

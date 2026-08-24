import { parseTzOffset } from '../timezone.js';

const pad = (n, len = 2) => String(n).padStart(len, '0');

export function formatTimestamp(utcSeconds, offsetMs, offsetLabel) {
  const date = new Date(utcSeconds * 1000 + offsetMs);

  const y = date.getUTCFullYear();
  const mo = pad(date.getUTCMonth() + 1);
  const d = pad(date.getUTCDate());
  const h = pad(date.getUTCHours());
  const mi = pad(date.getUTCMinutes());
  const s = pad(date.getUTCSeconds());

  return `${y}-${mo}-${d}T${h}:${mi}:${s}${offsetLabel}`;
}

export function attachCopyTimestamp(player, btnEl, anchorTime) {
  const tz = parseTzOffset(new URLSearchParams(location.search).get('tz')) || { offsetMs: 0, offsetLabel: '+00:00' };

  btnEl.addEventListener('click', () => {
    const totalSeconds = anchorTime + player.time();
    const timestamp = formatTimestamp(totalSeconds, tz.offsetMs, tz.offsetLabel);

    navigator.clipboard.writeText(timestamp).catch(() => {});
  });
}

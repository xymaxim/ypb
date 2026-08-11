import { MediaPlayer } from '../player.js';

const pad = (n, len = 2) => String(n).padStart(len, '0');

function parseTzOffset(raw) {
  const m = /^([+-])(\d{1,2})(?::([0-5]\d))?$/.exec(raw);
  if (!m) return null;
  const h = parseInt(m[2], 10);
  if (h > 23) return null;
  const min = m[3] ? parseInt(m[3], 10) : 0;
  const ms = (h * 3600 + min * 60) * 1000;
  return { offsetMs: m[1] === '-' ? -ms : ms, offsetLabel: `${m[1]}${pad(h)}:${pad(min)}` };
}

function formatTime(utcSeconds, offsetMs, offsetLabel) {
  if (!Number.isFinite(utcSeconds)) return '--';
  const date = new Date(utcSeconds * 1000 + offsetMs);

  const y = date.getUTCFullYear();
  const mo = pad(date.getUTCMonth() + 1);
  const d = pad(date.getUTCDate());
  const h = pad(date.getUTCHours());
  const mi = pad(date.getUTCMinutes());
  const s = pad(date.getUTCSeconds());

  return `${y}-${mo}-${d} ${h}:${mi}:${s} UTC${offsetLabel}`;
}

export function attachPlayheadDisplay(player, el, anchor) {
  const tz = parseTzOffset(new URLSearchParams(location.search).get('tz')) || { offsetMs: 0, offsetLabel: '+00:00' };
  const update = () => {
    el.textContent = formatTime(anchor + player.time(), tz.offsetMs, tz.offsetLabel);
  };
  player.on(MediaPlayer.events.PLAYBACK_TIME_UPDATED, update);
  player.on(MediaPlayer.events.PLAYBACK_SEEKED, update);
  update();
}

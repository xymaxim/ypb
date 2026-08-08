import { MediaPlayer } from '../player.js';

function formatTime(utcSeconds) {
  if (!Number.isFinite(utcSeconds)) return '--';
  const date = new Date(utcSeconds * 1000);
 
  const pad = (n, len = 2) => String(n).padStart(len, '0');
  const y = date.getUTCFullYear();
  const mo = pad(date.getUTCMonth() + 1);
  const d = pad(date.getUTCDate());
  const h = pad(date.getUTCHours());
  const mi = pad(date.getUTCMinutes());
  const s = pad(date.getUTCSeconds());
 
  return `${y}-${mo}-${d} ${h}:${mi}:${s} +00:00`;
}

export function attachPlayheadDisplay(player, el, anchor) {
  const update = () => {
    el.textContent = formatTime(anchor + player.time());
  };
  player.on(MediaPlayer.events.PLAYBACK_TIME_UPDATED, update);
  player.on(MediaPlayer.events.PLAYBACK_SEEKED, update);
  update();
}

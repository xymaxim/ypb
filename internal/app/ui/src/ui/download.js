import { formatTimestamp } from './timestamp.js';
import { parseTzOffset } from '../timezone.js';

function formatDuration(totalSeconds) {
  const s = Math.round(totalSeconds);

  const days = Math.floor(s / 86400);
  const hours = Math.floor((s % 86400) / 3600);
  const minutes = Math.floor((s % 3600) / 60);
  const seconds = s % 60;

  let out = '';
  if (days) out += `${days}d`;
  if (hours) out += `${hours}h`;
  if (minutes) out += `${minutes}m`;
  if (seconds || !out) out += `${seconds}s`;

  return out;
}

export function attachCopyDownload(
  btnEl,
  videoId,
  startTargetTime,
  endTargetTime,
) {
  const tz = parseTzOffset(new URLSearchParams(location.search).get('tz')) ||
    { offsetMs: 0, offsetLabel: '+00:00' };

  btnEl.addEventListener('click', () => {
    if (
      !videoId ||
      !Number.isFinite(startTargetTime) ||
      !Number.isFinite(endTargetTime)
    ) {
      return;
    }

    const start = formatTimestamp(startTargetTime, tz.offsetMs, tz.offsetLabel);
    const duration = formatDuration(endTargetTime - startTargetTime);
    const command = `ypb download -i ${start}/${duration} ${videoId}`;

    navigator.clipboard.writeText(command).catch(() => {});
  });
}

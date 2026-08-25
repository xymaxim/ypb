import { createPlayer, MediaPlayer } from './player.js';
import { attachPlayheadDisplay } from './ui/playhead.js';
import { attachQualitySelector, attachTrackSelector } from './ui/selectors.js';
import { attachTakeScreenshot } from './ui/screenshot.js';
import { attachCopyTimestamp } from './ui/timestamp.js';
import { attachCopyDownload } from './ui/download.js';

const interval = location.pathname.replace(/^\/+/, '') || 'now';
const mpd = new URL(`/mpd/${encodeURIComponent(interval)}`, location.href);
const params = new URLSearchParams(location.search);
const latency = params.get('latency') ?? params.get('l');
if (latency !== null) mpd.searchParams.set('latency', latency);
const mpdURL = mpd.href;

const video = document.getElementById('player');

const errorEl = document.getElementById('error');
const loadingEl = document.getElementById('loading');

const videoLink = document.getElementById('live');
const channelLink = document.querySelector('#channel a');

const playheadEl = document.getElementById('playhead');
const playBarEl = document.getElementById('play-bar');
const screenshotBtn = document.getElementById('screenshot');
const copyTimestampBtn = document.getElementById('copy-timestamp');
const copyDownloadBtn = document.getElementById('copy-download');

const qualitiesEl = document.getElementById('qualities');
const tracksEl = document.getElementById('tracks');

let videoId = '';
try {
  const res = await fetch('/info');
  if (res.ok) {
    const info = await res.json();
    videoId = info.id;
    document.title = info.title;
    videoLink.textContent = info.title;
    videoLink.href = `https://www.youtube.com/live/${info.id}`;
    if (info.channelId && info.channelTitle) {
      channelLink.textContent = info.channelTitle;
      channelLink.href = `https://www.youtube.com/channel/${info.channelId}`;
    }
  }
} catch {}

let startActualTime = NaN;
let startTargetTime = NaN;
let endTargetTime = NaN;
let manifestReady = false;

loadingEl.textContent = 'Rewinding...';
loadingEl.classList.remove('hidden');

try {
  const res = await fetch(mpdURL, { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    let detail = '';
    try {
      detail = (await res.text()).trim().replace(/^\d+\s*/, '');
    } catch {
      /* ignore */
    }
    throw new Error(
      detail
        ? `manifest request failed: ${res.status}: ${detail}`
        : `manifest request failed: ${res.status}`,
    );
  }
  const data = await res.json();
  startActualTime = new Date(data.metadata.startActualTime).getTime() / 1000;
  startTargetTime = new Date(data.metadata.startTargetTime).getTime() / 1000;
  if (data.metadata.endTargetTime) {
    endTargetTime = new Date(data.metadata.endTargetTime).getTime() / 1000;
  }
  manifestReady = true;
} catch (err) {
  errorEl.textContent = `Playback error: ${err.message || 'unknown'}`;
  loadingEl.classList.add('hidden');
}

if (manifestReady) {
  loadingEl.textContent = 'Loading...';
  const player = createPlayer(video, mpdURL);
  window.player = player;

  attachPlayheadDisplay(player, playheadEl, startActualTime);
  attachQualitySelector(player, qualitiesEl);
  attachTrackSelector(player, tracksEl);
  attachTakeScreenshot(player, video, screenshotBtn, videoId, startActualTime);
  attachCopyTimestamp(player, copyTimestampBtn, startActualTime);
  attachCopyDownload(copyDownloadBtn, videoId, startTargetTime, endTargetTime);

  player.on(MediaPlayer.events.ERROR, (e) => {
    loadingEl.classList.add('hidden');
    errorEl.textContent = `Playback error: ${e.error?.message || e.error || e.message || 'unknown'}`;
  });
  player.on(MediaPlayer.events.PLAYBACK_PLAYING, () => {
    loadingEl.classList.add('hidden');
    errorEl.textContent = '';
    playBarEl.classList.add('visible');
  });
  player.on(MediaPlayer.events.STREAM_INITIALIZED, () => {
    errorEl.textContent = '';
    copyDownloadBtn.classList.toggle('hidden', player.isDynamic());
  });
}

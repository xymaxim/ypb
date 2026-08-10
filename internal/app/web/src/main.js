import { createPlayer, MediaPlayer } from './player.js';
import { attachPlayheadDisplay } from './ui/playhead.js';
import { attachQualitySelector, attachTrackSelector } from './ui/selectors.js';
import { attachTakeScreenshot } from './ui/screenshot.js';

const interval = location.pathname.replace(/^\/+/, '') || 'now';
const mpdURL = new URL(`/mpd/${encodeURIComponent(interval)}`, location.href).href;

const video = document.getElementById('player');
const container = document.getElementById('player-container');

const statusEl = document.getElementById('status');

const videoLink = document.getElementById('live');
const channelLink = document.querySelector('#channel a');

const playheadEl = document.getElementById('playhead');
const playBarEl = document.getElementById('play-bar');
const screenshotBtn = document.getElementById('screenshot');

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
} catch {
}

let manifestSource;
let startActualTime = NaN;

try {
  const res = await fetch(mpdURL, { headers: { Accept: 'application/json' } });
  if (!res.ok) throw new Error(`manifest request failed: ${res.status}`);
  const data = await res.json();
  startActualTime = new Date(data.metadata.startActualTime).getTime() / 1000;
  manifestSource = URL.createObjectURL(
    new Blob([data.mpd], { type: 'application/dash+xml' })
  );
} catch (err) {
  statusEl.textContent = `Playback error: ${err.message || 'unknown'}`;
}

if (manifestSource) {
  const player = createPlayer(video, manifestSource);
  window.player = player;

  attachPlayheadDisplay(player, playheadEl, startActualTime);
  attachQualitySelector(player, qualitiesEl);
  attachTrackSelector(player, tracksEl);
  attachTakeScreenshot(player, video, screenshotBtn, videoId, startActualTime);

  player.on(MediaPlayer.events.ERROR, (e) => {
    statusEl.textContent = `Playback error: ${e.error?.message || e.error || e.message || 'unknown'}`;
  });
  player.on(MediaPlayer.events.PLAYBACK_PLAYING, () => {
    statusEl.textContent = '';
    playBarEl.classList.add('visible');
  });
  player.on(MediaPlayer.events.STREAM_INITIALIZED, () => { statusEl.textContent = ''; });
}

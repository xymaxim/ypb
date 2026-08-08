import { MediaPlayer } from '../player.js';

function renderList(el, label, items, activeIndex, onSelect) {
  el.innerHTML = '';

  const labelSpan = document.createElement('span');
  labelSpan.className = 'selector-label';
  labelSpan.textContent = `${label}:`;
  el.appendChild(labelSpan);

  items.forEach((item, index) => {
    const span = document.createElement('span');
    span.className = 'selector-item';
    span.textContent = item;
    span.classList.toggle('active', index === activeIndex);
    span.addEventListener('click', () => onSelect(index, span));
    el.appendChild(span);
  });
}

function markActive(el, index) {
  [...el.querySelectorAll('.selector-item')].forEach((item, i) => {
    item.classList.toggle('active', i === index);
  });
}

function qualityLabel(representation) {
  if (representation.height) return `${representation.height}p`;
  return 'auto';
}

export function attachQualitySelector(player, el) {
  let representations = [];
  let autoMode = true;

  const render = () => {
    representations = player.getRepresentationsByType('video');
    if (!representations.length) return;

    const current = player.getCurrentRepresentationForType('video');
    const currentIndex = representations.findIndex((r) => r.id === current?.id);
    const labels = ['Auto', ...representations.map(qualityLabel)];
    // index 0 is "Auto"; representations start at index 1
    renderList(el, 'Quality', labels, autoMode ? 0 : currentIndex + 1, (index) => {
      autoMode = index === 0;
      if (autoMode) {
        player.updateSettings({ streaming: { abr: { autoSwitchBitrate: { video: true } } } });
      } else {
        // Manual selection alone doesn't stop ABR from re-picking a
        // quality on the next segment, and it has to be disabled explicitly
        player.updateSettings({ streaming: { abr: { autoSwitchBitrate: { video: false } } } });
        player.setRepresentationForTypeByIndex('video', index - 1, true);
      }
      markActive(el, index);
    });
  };

  player.on(MediaPlayer.events.STREAM_INITIALIZED, render);

  player.on(MediaPlayer.events.QUALITY_CHANGE_RENDERED, () => {
    if (autoMode) markActive(el, 0);
  });
}

function trackLabel(track) {
  if (!track.codec) return 'unknown';
  // track.codec looks like: video/mp4;codecs="av01.0.00M.08"
  const index = track.codec.indexOf('codecs=');
  if (index === -1) return track.codec;
  return track.codec.slice(index + 'codecs='.length).replace(/"/g, '');
}

export function attachTrackSelector(player, el) {
  let tracks = [];

  const render = () => {
    tracks = player.getTracksFor('video');
    if (tracks.length < 2) return;
    const current = player.getCurrentTrackFor('video');
    const activeIndex = tracks.findIndex((t) => t.id === current?.id);
    renderList(el, 'Track', tracks.map(trackLabel), activeIndex, (index) => {
      player.setCurrentTrack(tracks[index]);
      markActive(el, index);
    });
  };

  player.on(MediaPlayer.events.STREAM_INITIALIZED, render);

  player.on(MediaPlayer.events.TRACK_CHANGE_RENDERED, (e) => {
    if (e.mediaType !== 'video' || !tracks.length) return;
    const index = tracks.findIndex((t) => t.id === e.newMediaInfo.id);
    if (index !== -1) markActive(el, index);
  });
}

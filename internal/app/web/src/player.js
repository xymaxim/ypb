import { MediaPlayer } from 'https://cdn.dashjs.org/v5.2.0/modern/esm/dash.all.min.js';

export function createPlayer(video, mpdURL) {
  const player = MediaPlayer().create();
  player.initialize(video, mpdURL, true);
  return player;
}

export { MediaPlayer };

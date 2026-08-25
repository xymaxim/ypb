import { MediaPlayer } from 'https://cdn.dashjs.org/v5.2.0/modern/esm/dash.all.min.js';

export function createPlayer(video, mpdURL) {
  const player = MediaPlayer().create();

  player.updateSettings({
    streaming: {
      delay: {
        useSuggestedPresentationDelay: false,
        liveDelay: 604800,
      },
      liveCatchup: {
        enabled: false,
      },
    },
  });

  // Force dash.js to skip initialization network requests. Since our media
  // segments are self-contained, injecting a dummy data URI instructs the
  // player to immediately begin fetching actual media segments.
  player.on(MediaPlayer.events.MANIFEST_LOADED, (e) => {
    const manifest = e.data;
    if (!manifest) return;
    (manifest.Period || []).forEach((period) => {
      (period.AdaptationSet || []).forEach((as) => {
        (as.Representation || []).forEach((rep) => {
          if (rep.SegmentTemplate) {
            rep.SegmentTemplate.initialization =
              'data:application/octet-stream;base64,';
          }
        });
      });
    });
  });

  player.on(MediaPlayer.events.ERROR, (event) => {
    console.error('Player Error:', event.message, 'Type:', event.error);
  });

  player.initialize(video, mpdURL, true);

  return player;
}

export { MediaPlayer };

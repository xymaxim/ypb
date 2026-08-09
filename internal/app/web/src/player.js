import { MediaPlayer } from 'https://cdn.dashjs.org/v5.2.0/modern/esm/dash.all.min.js';

export function createPlayer(video, mpdURL) {
    const player = MediaPlayer().create();

    // Force dash.js to skip initialization network requests. Since our media
    // segments are self-contained, injecting a dummy data URI instructs the
    // player to immediately begin fetching actual media segments.
    player.on(dashjs.MediaPlayer.events.MANIFEST_LOADED, function (e) {
        const manifest = e.data;
        if (!manifest) return;
        (manifest.Period || []).forEach(function (period) {
        (period.AdaptationSet || []).forEach(function (as) {
            (as.Representation || []).forEach(function (rep) {
                if (rep.SegmentTemplate) {
                    rep.SegmentTemplate.initialization = 'data:application/octet-stream;base64,';
                }
            });
        });
        });
    });

    player.on(MediaPlayer.events.ERROR, function (event) {
         console.error("Player Error:", event.message, "Type:", event.error);
    });
    
    player.initialize(video, mpdURL, true);
    
    return player;
}

export { MediaPlayer };

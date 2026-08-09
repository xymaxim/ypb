import { MediaPlayer } from 'https://cdn.dashjs.org/v5.2.0/modern/esm/dash.all.min.js';

export function createPlayer(video, mpdURL) {
    const player = MediaPlayer().create();

    // Extend the RequestModifier to intercept initialization requests
    player.addRequestInterceptor(function (request) {
        if (request.type === 'InitializationSegment') {
            request.url = 'data:application/octet-stream;base64,';
            request.withCredentials = false;
        }
        return Promise.resolve(request);
    });

    // // Clear out player tracking that depends on distinct init boundaries
    // player.updateSettings({
    //     streaming: {
    //         buffer: {
    //             fastSwitchEnabled: false // Prevents track switches from demanding unique init chunks
    //         }
    //     }
    // });

    player.on(MediaPlayer.events.ERROR, function (event) {
       console.error("Player Error:", event.message, "Type:", event.error);
    });
    
    player.initialize(video, mpdURL, true);
    
  return player;
}

export { MediaPlayer };

# Overview

Ypb runs in three modes: serve, download, and play.

All modes relies on [yt-dlp](https://github.com/yt-dlp/yt-dlp/) for fetching
video information and solving JavaScript challenges.

## Serve

Serve mode runs a local HTTP proxy server that handles [API](./reference/api.md)
requests to locate moments in the stream, generate MPEG-DASH manifests (MPDs),
and serve media segments with HTTP error retry handling. The generated manifests
can be passed to any MPEG-DASH compatible player or downloader.

## Download

Download mode saves excerpts to local files with a single command. It composes a
manifest, then starts the proxy to stream segments before passing it to yt-dlp's
general extractor.

```mermaid
sequenceDiagram
    autonumber
    participant D as Download
    participant E as yt-dlp<br/>(extractor)
    participant P as Proxy
    participant Y as YouTube
    D->>D: Generate MPD<br/>(proxied based URLs)
    D->>E: Pass MPD
    loop Download
        E->>P: Request segment (proxied URL)
        P->>Y: Request segment
        Y-->>P: Stream segment
        P-->>E: Stream segment
    end
    E->>E: Write to file
```

## Play

Play mode is similar to serve mode, additionally serving a minimal web page with the [dash.js](https://dashjs.org/) player. The player talks to the proxy to fetch manifests and stream segments, letting you rewatch past moments in the browser.

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser<br/>(dash.js)
    participant P as Proxy
    participant Y as YouTube

    B->>P: Request MPD
    P->>P: Generate MPD
    P-->>B: MPD (proxied base URLs)
    loop Playback
        B->>P: Request segment (proxied URL)
        P->>Y: Request segment
        Y-->>P: Stream segment
        P-->>B: Stream segment
    end
```

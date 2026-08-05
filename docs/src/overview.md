# Overview

Ypb runs in two modes: serve and download.

```mermaid
sequenceDiagram
    autonumber
    participant D as Download
    participant S as Serve
    participant Y as YouTube
    participant E as yt-dlp<br/>(extractor)
    S->>Y: Fetch info (via yt-dlp)
    Y-->>S: Video info
    D->>S: Request MPD
    S->>S: Generate MPD
    S-->>D: MPD (proxied base URLs)
    D->>E: Pass MPD
    loop Download
        E->>S: Request segment (proxied URL)
        S->>Y: Request segment
        Y-->>S: Stream segment
        S-->>E: Stream segment
    end
    E->>E: Write to file
```

Serve mode runs a local HTTP proxy server that handles [API
requests](https://xymaxim.github.io/ypb/reference/api.html) to locate moments in
the stream, generate MPEG-DASH manifests (MPDs), and serve media segments with
HTTP error retry handling. It relies on yt-dlp for fetching video information
and solving JavaScript challenges.

Download mode saves excerpts to local files with
a single command, using the same proxy internally to compose manifests before
passing them to yt-dlp's general extractor.


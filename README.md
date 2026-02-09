# Ypb — A playback for YouTube live streams

*Rewind to past moments in live streams and play or download excerpts*

Ypb is a playback tool for YouTube live streams written in Go. It provides
MPEG-DASH access to past moments in live streams, allowing you to rewind beyond
the web player's limits, play selected excerpts instantly in any compatible
player, or download them as local files.

## Features

- Standalone CLI and proxy streaming server for playback
- Rewind precisely to past moments far beyond the web player’s limits
- Play excerpts immediately without downloading
- Works with any MPEG-DASH compatible player or downloader
- Leverages `yt-dlp` for reliable video info extraction and downloading

## Overview

Ypb can run in two modes: serve or download. Run the streaming proxy server
and make rewind requests to compose static or dynamic MPEG-DASH manifests, or
download excerpts to local files with a single command.

## Installation

Ypb works on Linux, macOS, and Windows.

Read the [Installation](https://xymaxim.github.io/ypb/guides/install/install.md)
guide for different ways to install and run `ypb`.

## Showcase

### Download stream excerpt

Download the latest 10 minutes from a live stream to a local file:

```shell
$ ypb download --interval 10m/now Mm_zVDDUeNA && ls
Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T054630+00_10m.f137.mp4
``` 

Or download a similar 10-minute excerpt from one day ago:

```shell
$ ypb download --interval now-1d10m/now-1d Mm_zVDDUeNA && ls
Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260207T054630+00_10m.f137.mp4
``` 

### Preview stream excerpt

Start the playback server to enable rewind requests:

```shell
$ ypb serve --port 8080 Mm_zVDDUeNA
```

With the server running, you can rewind to excerpts and play them, for example,
with `ffplay`:

```shell
$ ffplay -protocol_whitelist file,http,https,tcp,tls \
      http://localhost:8080/rewind/30m--now
```

## License

MIT.

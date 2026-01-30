# ypb

A playback for YouTube live streams.

*Rewind to past moments in live streams and play or download excerpts*

Ypb is a playback tool for YouTube live streams written in Go. It provides
MPEG-DASH access to past moments in live streams, letting you jump back beyond
the web player's limits, play selected excerpts instantly in any compatible
player, or download them as local files.

## Features

- **Standalone CLI and streaming proxy**: CLI tool for direct downloads and proxy
  streaming server for playback
- **Time-shift playback**: Precisely rewind to past moments in live streams
- **Direct streaming**: Play excerpts immediately without downloading
- **Standard compatibility**: Works with any MPEG-DASH compatible player or
  downloader
- **yt-dlp integration**: Leverages yt-dlp for reliable video info extraction
  and downloading

## Quick start

### Download stream excerpt

Download the latest 30 minutes from a live stream to a local file:

```bash
$ ypb download --interval 30m/now <video-id> && ls
Stream-Title_abcdefgh123_20260102T102030+00_30m.mp4
``` 

Or download the same 30-minute excerpt, but from one day ago:

```bash
$ ypb download --interval now-1d30m/now-1d <video-id> && ls
Stream-Title_abcdefgh123123_20260101T102030+00_30m.mp4
``` 

### Preview stream excerpt

Start the playback server to enable rewind requests:

```bash
$ ypb serve --port 8080 <video-id>
```

With the server running, you can rewind excerpts and play them, for example, with ffplay:

```bash
$ ffplay -protocol_whitelist file,http,https,tcp,tls \
      http://localhost:8080/rewind/30m--now
```

## License

MIT.

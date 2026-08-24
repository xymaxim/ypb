# Serve stream excerpts

In this tutorial, you'll start a playback server for a YouTube live stream.
The server gives you MPEG-DASH access to excerpts of the stream. You'll
rewind to a past moment with a single request, then send the result to a
media player or a downloader.

We'll use the [Northern Royal Albatross nesting
cam](https://www.youtube.com/live/Mm_zVDDUeNA) as our example.

!!! info ""
    :lucide-glasses: &nbsp; Want some background first? Read [Core
    concept](../overview.md#core-concept) in Overview.

## Prerequisites

Before you begin, make sure you have:

1. `ypb` [installed](../guides/install/install.md)
2. `ffmpeg` (for `ffplay`) and `yt-dlp`, if you want to follow the playback and
   download steps

## Start the playback server

Run `ypb serve` with the YouTube video ID:

```shell
$ ypb serve Mm_zVDDUeNA
(<<) Stream 'Live & Just Hatched! Royal Albatross Cam - NZ Dept. of Conservation | Cornell Lab' is alive!
(<<) Playback started and listening on http://localhost:9000...
```

The server keeps running in this terminal. Open a second terminal for the
next steps.

## Rewind to a past moment

The playback server does one job: rewind to the moment you ask for. It finds
that moment, builds an MPEG-DASH manifest for it, and streams the segments
the manifest points to. You ask for a moment through the `/mpd/{interval}`
endpoint, where `{interval}` is the moment you want. Let's ask for the last
30 minutes:

    curl http://localhost:9000/mpd/30m--now

You should see a wall of XML. That's the manifest describing the excerpt.

The `{interval}` part uses the same format as the `-i/--interval` option of
the `download` command. Two things change in a URL: use `--` instead of `/`,
and avoid whitespace (or percent-encode it).

> See [Specifying the rewind
> time](../reference/cli.md#specifying-the-rewind-time) for all the
> interval formats.

!!! note "Static and dynamic manifests"
    The kind of manifest you get depends on the interval. A closed
    interval, one with an end, gives you a static manifest: it describes a
    finished excerpt and never changes. An open interval, one with no end,
    gives you a dynamic manifest: it keeps growing as the stream goes on.

    ```text
    # Static: excerpts with a fixed end
    curl localhost:9000/mpd/30m--now
    curl localhost:9000/mpd/12:00--30m

    # Dynamic: excerpts with no end
    curl localhost:9000/mpd/now
    curl localhost:9000/mpd/12:00
    ```

## Play the excerpt

Now let's play the excerpt with a player, without downloading it first. This
tutorial uses mpv. Point it at the same URL:

```shell
mpv --demuxer-lavf-o=protocol_whitelist=file,http,https,tcp,tls \
  localhost:9000/mpd/30m--now
```

The `--demuxer-lavf-o` option passes the protocol whitelist to mpv's FFmpeg
demuxer, letting it open the manifest and fetch its media segments.

!!! tip "Using a different player"
    Any MPEG-DASH compatible player works here, not just mpv. For example,
    [ffplay](https://ffmpeg.org/ffplay.html) and
    [VLC](https://www.videolan.org/vlc/) can both open the same manifest URL
    directly.

## Download the excerpt

You can also point the manifest to a downloader, instead of a player.

For example, let's use yt-dlp's general extractor directly: this is kind of
similar to what the [`download`](../reference/cli.md#download) command does
under the hood:

```shell
yt-dlp -o output.mp4 http://localhost:9000/mpd/30m--now
```

!!! tip "Using a different downloader"
    Other downloaders work too: [FFmpeg](https://www.ffmpeg.org/), GPAC's
    [MP4Box](https://github.com/gpac/gpac/wiki/MP4Box/), or
    [dash-mpd-cli](https://emarsden.github.io/dash-mpd-cli/).

## See also

- [Play a stream in the browser](../guides/play-stream.md) — rewind using the
  built-in web player, the friendlier way
- [Download raw media segments](../guides/download-segments.md) — download the
  individual segments for your own processing

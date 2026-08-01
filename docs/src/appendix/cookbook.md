# Cookbook

## List available formats with yt-dlp

Before choosing which formats to download, it helps to see what's
available. Every YouTube's MPEG-DASH representation has an itag value that
identifies it. For a playing live stream, you can find these in the browser's
*Stats for nerds* popup. To show all available formats, running yt-dlp with `-F`
is helpful:

```shell
$ yt-dlp --live-from-start -F STREAM
ID  EXT  RESOLUTION FPS CH │    TBR PROTO │ VCODEC         VBR ACODEC      ABR ASR MORE INFO
───────────────────────────────────────────────────────────────────────────────────────────────────
140 m4a  audio only      2 │   144k dashG │ audio only         mp4a.40.2  144k 44k medium, m4a_dash
160 mp4  256x144     15    │   212k dashG │ avc1.42c00b   212k video only          144p, mp4_dash
...
137 mp4  1920x1080   30    │  5019k dashG │ avc1.640028  5019k video only          1080p, mp4_dash
248 webm 1920x1080   30    │  2896k dashG │ vp9          2896k video only          1080p, webm_dash
```

## Play a saved manifest with mpv

mpv can play a saved MPEG-DASH manifest (MPD) directly, without
downloading anything first.

The manifest's `SegmentTemplate`s reference `http://localhost:9000/...`
segments, but by default ffmpeg's demuxer only allows the `file` and
`crypto` protocols for local files. Whitelist the ones the manifest needs:

```
mpv --demuxer-lavf-o='protocol_whitelist="file,crypto,data,http,https,tcp,tls"' \
  manifest.mpd
```

To pick a specific stream, use mpv's interactive track selector: press `g`
then `v` for video, or `g` then `a` for audio. Each opens a console menu
listing the available tracks to choose from.

## Pass the output name to external programs

`ypb serve`'s JSON response includes `metadata.outputName`, the same name
`ypb download` would use for the same input. Reuse it when running another
tool against the stream:

```shell
$ ypb serve STREAM
$ response=$(curl -s -H 'Accept: application/json' http://localhost:9000/mpd/10s--now)
$ filename=$(printf '%s' "$response" | jq -r '.metadata.outputName')
```

yt-dlp can read the manifest off disk instead of fetching it again, so save it
and pass `filename` for the output location:

```shell
$ printf '%s' "$response" | jq -r '.mpd' > /tmp/manifest.mpd
$ yt-dlp --enable-file-urls -o "${output_name}.%(ext)s" "file:///tmp/manifest.mpd"
```

## Download multiple formats from one saved manifest

Requesting `/mpd` from `ypb serve` lets you rewind once and save the manifest, then run
`yt-dlp` against it as many times as you like. Each run reads the same
saved segments instead of triggering a new rewind, as `ypb download` would:

```shell
$ ypb serve STREAM
$ response=$(curl -s -H 'Accept: application/json' http://localhost:9000/mpd/10s--now)
$ printf '%s' "$response" | jq -r '.mpd' > /tmp/manifest.mpd

$ yt-dlp --enable-file-urls -o "output.f%(format_id)s.%(ext)s" -f 137 "file:///tmp/manifest.mpd"
$ yt-dlp --enable-file-urls -o "output.f%(format_id)s.%(ext)s" -f 140 "file:///tmp/manifest.mpd"
```

You could get the same files with a single `-f "137,140"` call, but keeping the
manifest around lets you work incrementally.

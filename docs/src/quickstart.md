# Get started with ypb

This tutorial shows how to install `ypb` and demonstrates its main usage
scenarios.

You will explore two running modes: download and serve. We will download a live
stream excerpt, explaining the main command options. Next, we will see how to
play past moments without downloading.

## Installation

There are two ways to get `ypb` running: install from pre-built binaries or run in
a container. The choice depends on what's already installed on your system and
your preferences.

### Install from binaries

Ypb requires `yt-dlp` and the [related
dependencies](guides/install/install.md#requirements). If you already have a
working `yt-dlp` installation on your computer (ensure it is in your `PATH`),
the quickest way to start is to download the pre-built binaries.

Use the links from the [latest
release](https://github.com/xymaxim/ypb/releases/latest) below for your platform
and architecture:

|       | Linux                 | macOS                  | Windows                 |
|-------|-----------------------|------------------------|-------------------------|
| AMD64 | [ypb-linux-amd64.zip] | [ypb-darwin-amd64.zip] | [ypb-windows-amd64.zip] |
| ARM64 | [ypb-linux-arm64.zip] | [ypb-darwin-arm64.zip] | [ypb-windows-arm64.zip] |

[ypb-linux-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-linux-amd64.zip
[ypb-linux-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-linux-arm64.zip
[ypb-darwin-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-darwin-amd64.zip
[ypb-darwin-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-darwin-arm64.zip
[ypb-windows-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-windows-amd64.zip
[ypb-windows-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-windows-arm64.zip

Download and unzip a file to your working directory.

Verify the version with the following command:

**Linux/macOS**

    chmod +x ypb && ./ypb version
    
**Windows**

(Type `ypb.exe` instead of `ypb`; no installation needed.)

    .\ypb.exe version

> See [Pre-built binaries](guides/install/binaries.md) for more details.

### Try in a container

Running in a container allows you to try `ypb` in an isolated environment with all required dependencies pre-installed.

**Prerequisites:** [Podman](https://podman.io/getting-started/installation) or [Docker](https://docs.docker.com/get-docker/)

**macOS/Windows only:** Initialize the Podman machine (one-time setup):

    podman machine init && podman machine start

Pull the container image and verify the version:

    podman pull ghcr.io/xymaxim/ypb
    podman run --rm ghcr.io/xymaxim/ypb version

To run commands with access to your current directory:

    podman run --rm -v .:/content ghcr.io/xymaxim/ypb ...

> See [Container image](guides/install/container.md) for creating [recommended
> aliases](guides/install/container.md#recommended-aliases) and more details.

## Download excerpts

Let’s start by downloading a small excerpt from a live stream.

If you are not sure what to watch, the [Cornell Lab Bird
Cams](https://www.allaboutbirds.org/cams/) project provides access to beatiful
bird cam streams across the world. As an example, let's watch at the [Northern
Royal Albatross nesting](https://www.allaboutbirds.org/cams/royal-albatross/) at
Taiaroa Head, New Zealand.

To get the last one-minute excerpt from the [YouTube
stream](https://www.youtube.com/live/Mm_zVDDUeNA), run this command by providing
the rewind interval (``-i/--interval``) and YouTube video ID (required
argument):

```shell
$ ypb download --interval 1m/now Mm_zVDDUeNA
(<<) Collecting info about https://www.youtube.com/live/Mm_zVDDUeNA...
WARNING: [youtube] No supported JavaScript runtime could be found. Only deno is
enabled by default; to use another runtime add --js-runtimes RUNTIME[:PATH] to
your command/config. YouTube extraction without a JS runtime has been
deprecated, and some formats may be missing. See
https://github.com/yt-dlp/yt-dlp/wiki/EJS for details on installing one
Stream 'Live & Just Hatched! Royal Albatross Cam - NZ Dept. of Conservation | Cornell Lab' is alive!

(<<) Locating start and end moments...
Actual start: Sun, 08 Feb 2026 11:22:50 +0000 (-2s), sq=1647523
  Actual end: Sun, 08 Feb 2026 11:23:55 +0000, sq=1647535

(<<) Downloading and merging media...
yt-dlp: [generic] Extracting URL: http://localhost:8080/mpd
yt-dlp: [generic] mpd: Downloading webpage
yt-dlp: WARNING: [generic] Falling back on generic information extractor
yt-dlp: [generic] mpd: Extracting information
yt-dlp: [info] mpd: Downloading 1 format(s): 137+140
yt-dlp: [dashsegments] Total fragments: 13
yt-dlp: [download] Destination: Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T112250+00_1m.f137.mp4
yt-dlp: [download] 100.0% of ~   1.00KiB at      0.00B/s ETA Unknown (frag 0/13)
...
yt-dlp: [Merger] Merging formats into "Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T112250+00_1m.mp4"
yt-dlp: Deleting original file Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T112250+00_1m.f137.mp4 (pass -k to keep)
yt-dlp: Deleting original file Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T112250+00_1m.f140.m4a (pass -k to keep)
```

> [!WARNING]
> You may see warnings about a missing JavaSript runtime, if you have not
> installed or enabled it:
>
> ```shell
> WARNING: [youtube] No supported JavaScript runtime could be found. Only deno is
> enabled by default; to use another runtime add --js-runtimes RUNTIME[:PATH] to
> your command/config. YouTube extraction without a JS runtime has been
> deprecated, and some formats may be missing. See
> https://github.com/yt-dlp/yt-dlp/wiki/EJS for details on installing one
> ```
> 
> You may also get HTTP 403 errors (approximately every 30 seconds) during the
> download:
> 
> ```shell
> time=2026-02-08T11:23:05.127+00:00 level=WARN msg="got transient HTTP error,
> retrying" status=403 method=GET url=...
> ```
>
> While this currently works by retrying and collecting video info again, it is
> highly recommended to set up the [additional
> dependencies](guides/install/install.md#additional-dependencies) to help avoid
> such errors.

As you can see, downloading consists of three steps: (1) collecting video
information, (2) locating start and end moments, and (3) the download itself
with audio and video merging at the end. The first and third stages are carried
out by `yt-dlp`.

Once the download finished, a single MP4 file can be found in the current
working directory:

    Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260208T112250+00_1m.mp4

### Specify formats

By default, we let `yt-dlp` choose the audio and video formats automatically (it
selects the best available quality). This gives you the flexibility to use
familiar `yt-dlp` options or even your existing configuration file.

You can pass options directly to `yt-dlp` by adding them after the `--`
separator.

> See [Pass options to yt-dlp](reference/cli.md#ytdlp-options) for more details
> and examples.

For example, let's use the `yt-dlp`'s `-f` option to download only the best
quality audio:

```shell
$ ypb download -i 30s/now Mm_zVDDUeNA -- -f bestaudio -x
```

Running `yt-dlp -F ...` can be helpful here to list all available formats:

``` shell
$ yt-dlp -F --live-from-start Mm_zVDDUeNA
[youtube] Extracting URL: Mm_zVDDUeNA
[youtube] Mm_zVDDUeNA: Downloading webpage
[youtube] Mm_zVDDUeNA: Downloading android sdkless player API JSON
[youtube] Mm_zVDDUeNA: Downloading web safari player API JSON
[youtube] Mm_zVDDUeNA: Downloading MPD manifest
[info] Available formats for Mm_zVDDUeNA:
ID  EXT RESOLUTION FPS │   TBR PROTO │ VCODEC        VBR ACODEC      ABR ASR MORE INFO
─────────────────────────────────────────────────────────────────────────────────────────────────
139 m4a audio only     │   64k dashG │ audio only        mp4a.40.5   64k 22k DASH audio, m4a_dash
140 m4a audio only     │  144k dashG │ audio only        mp4a.40.2  144k 44k DASH audio, m4a_dash
160 mp4 256x144     15 │  212k dashG │ avc1.42c00b  212k video only          DASH video, mp4_dash
133 mp4 426x240     30 │  456k dashG │ avc1.4d4015  456k video only          DASH video, mp4_dash
134 mp4 640x360     30 │ 1008k dashG │ avc1.4d401e 1008k video only          DASH video, mp4_dash
135 mp4 854x480     30 │ 1350k dashG │ avc1.4d401f 1350k video only          DASH video, mp4_dash
136 mp4 1280x720    30 │ 2684k dashG │ avc1.4d401f 2684k video only          DASH video, mp4_dash
137 mp4 1920x1080   30 │ 5019k dashG │ avc1.640028 5019k video only          DASH video, mp4_dash
```

> See `yt-dlp`'s [Format
> selection](https://github.com/yt-dlp/yt-dlp?tab=readme-ov-file#format-selection)
> for the option syntax and some examples.

### Specify start and end

The interval start and end moments supports flexible formats including date and
times, durations, keywords, and even time arithmetic expressions.

The local time in New Zealand is UTC+12 or UTC+13 during daylight saving, and it
might be nighttime on the stream depending on your location. For example, let's
see what's on the stream at noon:

``` shell
# If it is already noon there
$ ypb download -i '12:00+13/1m' Mm_zVDDUeNA

# Or noon yesterday
$ ypb download -i '12:00+13 - 1d/1m' Mm_zVDDUeNA
```

> See [Specifying the rewind
> interval](reference/cli.md#specifying-the-rewind-interval) for the reference
> on interval part formats.

## Serve and play excerpts

Now let's explore another feature: play without downloading.

### Start a playback server

This requires us to start a playback in serve mode:

```shell
$ ypb serve Mm_zVDDUeNA
(<<) Served started and listening on http://localhost:8080
```

As you see, we are not using the interval option here. Format selection is also
not applicable. The playback server is now running and waiting for our requests.

### Send rewind requests

To rewind an excerpt, open another terminal and type:

    curl localhost:8080/mpd/30m--now

This should return the raw content of the composed static MPEG-DASH manifest.

The rewind path parameter `/mpd/{interval}` has the same format as the
`-i/--interval` option except that it should be properly URL escaped: use `--`
instead of `/`, avoid whitespaces or use percent encoding.

> See the [API](reference/api.md) reference for available endpoints.

### Play stream excerpts

Now the intriguing part: playing the excerpt.

``` shell
ffplay -protocol_whitelist file,http,https,tcp,tls \
  localhost:8080/mpd/30m--now
```

The option `-protocol_whitelist` is required to allow `ffplay` openining the
manifest and fetching media segments.

However, the choice is not limited to `ffplay`: you can use any MPEG-DASH
compatible player you prefer.

### Download media

As a bonus, let's see how to download media content from the composed manifest.

This is actually almost how `ypb download` works behind the scenes:

    yt-dlp -o output.mp4 http://localhost:8080/mpd/30m--now

> Other downloader options: [FFmpeg](https://www.ffmpeg.org/), GPAC's
> [MP4Box](https://github.com/gpac/gpac/wiki/MP4Box/), or
> [dash-mpd-cli](https://emarsden.github.io/dash-mpd-cli/).

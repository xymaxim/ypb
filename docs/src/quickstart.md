# Quickstart

This tutorial shows how to install `ypb` and demonstrates its main usage
scenarios.

First, we will watch a live stream in the browser and rewind to past moments
without downloading. Next, we will download a selected stream excerpt to a local
file.

## Installation

There are two ways to get `ypb` running: install from pre-built binaries or run in
a container. The choice depends on what's already installed on your system and
your preferences.

### Install from binaries

Ypb requires `yt-dlp` and the [related
dependencies](guides/install/install.md). If you already have a
working `yt-dlp` installation on your computer (ensure it is in your `PATH`),
the quickest way to start is to download the pre-built binaries.

Use the links from the [latest
release](https://github.com/xymaxim/ypb/releases/latest) below for your platform
and architecture:

*Latest release {{git.short_tag}}*

<div class="grid cards three-cols" markdown>

-   :material-linux:{ .lg } __Linux__

    [amd64][ypb-{{ git.short_tag }}-linux-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-linux-arm64.zip]

-   :material-apple:{ .lg } __macOS__

    [amd64][ypb-{{ git.short_tag }}-darwin-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-darwin-arm64.zip]
    
-   :material-microsoft-windows:{ .lg } __Windows__

    [amd64][ypb-{{ git.short_tag }}-windows-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-windows-arm64.zip]

</div>

[ypb-{{ git.short_tag }}-linux-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-linux-amd64.zip
[ypb-{{ git.short_tag }}-linux-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-linux-arm64.zip
[ypb-{{ git.short_tag }}-darwin-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-darwin-amd64.zip
[ypb-{{ git.short_tag }}-darwin-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-darwin-arm64.zip
[ypb-{{ git.short_tag }}-windows-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-windows-amd64.zip
[ypb-{{ git.short_tag }}-windows-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-windows-arm64.zip

Download and unzip a file to your working directory.

Verify the version with the following command:

=== "Linux/macOS"

    ```shell
    chmod +x ypb && ./ypb version
    ```

=== "Windows"

    ```shell
    .\ypb.exe version
    ```

!!! important "Update to nightly"
    Make sure to update `yt-dlp` to the nightly build:

        yt-dlp --update-to nightly

<div class="grid" markdown>

See the full [Pre-built binaries](guides/install/binaries.md) installation guide for more details.
{ .card }

</div>

!!! warning
    You may see warnings about a missing JavaScript runtime, HTTP 403 errors
    (about every 30 seconds), or a "Sign in to confirm you're not a bot" error
    when cookies are missing. See [Setup](guides/install/binaries.md#setup) for
    how to avoid these.

### Try in a container

Running in a container allows you to try ypb in an isolated environment with all required dependencies pre-installed.

**Prerequisites:** [Podman](https://podman.io/getting-started/installation) or [Docker](https://docs.docker.com/get-docker/), with Compose

**macOS/Windows only, Podman:** Initialize the Podman machine (one-time setup):

    podman machine init && podman machine start

Pull the compose file and extract it to a local directory:

    podman artifact pull ghcr.io/xymaxim/ypb-compose
    podman artifact extract ghcr.io/xymaxim/ypb-compose ~/ypb-app
    cd ~/ypb-app

Verify the version:

    podman compose run --rm ypb version

!!! warning
    The container already includes a JavaScript runtime and a PO token provider, so the only
    thing left to set up is
    [cookies](guides/install/container.md#set-up-cookies-recommended). Without
    them, you may see a "Sign in to confirm you're not a bot" error.

<div class="grid" markdown>

See the full [Container](guides/install/container.md) installation guide for more details.
{ .card }

</div>

## Play a stream in the browser

Let's start by watching a stream without downloading.

If you are not sure what to watch, the [Cornell Lab Bird
Cams](https://www.allaboutbirds.org/cams/) project provides access to beautiful
bird cam streams across the world. As an example, let's watch the [Northern
Royal Albatross nesting](https://www.allaboutbirds.org/cams/royal-albatross/) at
Taiaroa Head, New Zealand.

### Run the player

Ypb includes a built-in web player. Run it for the
[stream](https://www.youtube.com/live/Mm_zVDDUeNA) by providing its YouTube video
ID:

```
$ ypb play Mm_zVDDUeNA
(<<) Stream 'Northern Royal Albatross Cam - NZ Dept. of Conservation #RoyalCam | Cornell Lab' is alive!
(<<) Playback started and listening on http://localhost:9000...
:::: Open http://localhost:9000/now in your browser to play
```

Now open http://localhost:9000/now in your browser. By default the player shows
the live edge of the stream.

### Rewind to a past moment

To jump to a specific moment, add it to the path.

The local time in New Zealand is UTC+12 or UTC+13 during daylight saving, and it
might be nighttime on the stream depending on your location. For example, let's
see what's on the stream at noon:

    # If it's already noon there
    http://localhost:9000/12:00+13

    # Or noon yesterday
    http://localhost:9000/12:00+13-1d

!!! info "Rewind precision"
    The moment you rewind to is snapped to the nearest media segment, so the
    actual time can differ a bit from the requested one. See [Why does the
    actual time differ from the target
    time?](appendix/questions.md#why-does-the-actual-time-differ-from-the-target-time)
    for details.

!!! example "Moment format examples"
    The requested moment supports flexible formats: dates and times, durations,
    keywords like `now`, and time arithmetic expressions.

    - Full date and time, with a timezone offset: `2026-01-02T10:20:30+00`
    - Time of the current day, in the local time zone: `10:20`
    - Relative to now (30 minutes ago): `now-30m`

    !!! info ""
        See [Specifying the rewind
        interval](reference/cli.md#specifying-the-rewind-interval) for the full
        reference on interval part formats.

### Preview an excerpt

To play an excerpt instead of continuing live, add an end moment to the path.

For example, a 10-minute excerpt from today's noon:

```text
http://localhost:9000/12:00+13--10m
```

!!! example "Interval format examples"
    A bounded excerpt combines a start and an end, separated by `--`:

    - Between two dates and times:
      `2026-01-02T10:20:30+00--2026-01-02T10:25:30+00`
    - Start time plus a duration:
      `2026-01-02T10:20:30+00--5m`
    - The last 30 minutes up to now:
      `30m--now`

Once the excerpt is loaded, you can quickly seek within it using the seek bar.
The preview is also a way to fine-tune the interval before downloading: click
**Copy the download command** button (`D`) to copy a `ypb download` command for
the current interval.

For the excerpt above, it would look like this:

```shell
ypb download -i 2026-08-18T12:00:00+13/10m Mm_zVDDUeNA
```

<div class="grid cards" markdown>

- **Learn more**

    ---
    
    See the full guide for other player features, including setting the output
    timezone, correcting for streaming latency, and more.
    
    :octicons-arrow-right-24: [Read the player guide](guides/play-stream.md)

</div>

## Download an excerpt

Once you've found an interesting moment, you can save it to a local file.

### Run the download

Let's download the excerpt we just previewed above. Paste the command you copied
and run it:

```shell
$ ypb download -i 2026-08-18T12:00:00+13/10m Mm_zVDDUeNA
(<<) Collecting info about https://www.youtube.com/live/Mm_zVDDUeNA...
Stream 'Live & Just Hatched! Royal Albatross Cam - NZ Dept. of Conservation | Cornell Lab' is alive!
(<<) Locating start and end moments...
Actual start: Mon, 17 Aug 2026 23:00:05 +0000 (-2s), sq=1720173
  Actual end: Mon, 17 Aug 2026 23:10:07 +0000, sq=1720185
(<<) Downloading and merging media...
yt-dlp: [generic] Extracting URL: http://localhost:9000/mpd
yt-dlp: [generic] mpd: Downloading webpage
yt-dlp: WARNING: [generic] Falling back on generic information extractor
yt-dlp: [generic] mpd: Extracting information
yt-dlp: [info] mpd: Downloading 1 format(s): 137+140
yt-dlp: [dashsegments] Total fragments: 130
yt-dlp: [download] Destination: Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260817T230005+00_10m.f137.mp4
yt-dlp: [download] 100.0% of ~  10.00MiB at   12.34MiB/s ETA Unknown (frag 0/130)
yt-dlp: [Merger] Merging formats into "Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260817T230005+00_10m.mp4"
yt-dlp: Deleting original file Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260817T230005+00_10m.f137.mp4 (pass -k to keep)
yt-dlp: Deleting original file Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260817T230005+00_10m.f140.m4a (pass -k to keep)
```

As you can see, downloading consists of three steps: (1) collecting video
information, (2) locating start and end moments, and (3) the download itself
with audio and video merging at the end. The first and third stages are carried
out by yt-dlp.

Once the download finished, a single MP4 file can be found in the current
working directory:

    Live-and-Just-Hatched-Royal_Mm_zVDDUeNA_20260817T230005+00_10m.mp4

> :octicons-arrow-right-24: See [Specifying the rewind
> interval](reference/cli.md#specifying-the-rewind-interval) for the accepted
> start and end formats.

### Choose audio and video formats

By default, we let yt-dlp choose the audio and video formats automatically,
following its own defaults or any preferences set in your yt-dlp's
[configuration file](https://github.com/yt-dlp/yt-dlp#configuration).

!!! tip "Configuring yt-dlp"
    Ypb uses yt-dlp for specific tasks, like fetching video info and
    downloading, so its configuration file also applies to ypb. Add options
    there (formats, cookies, and more) to apply them to every download. See how
    to set it up for your install: [Pre-built
    binaries](guides/install/binaries.md#configure-yt-dlp) or [Container
    (Compose)](guides/install/container.md#configure-yt-dlp).
    
Alternatively, you can pass options directly to yt-dlp by adding them after
the `--` separator (see [Passing options to
yt-dlp](reference/cli.md#passing-options-to-yt-dlp) for more details). For
example, let's use the yt-dlp's `-f` option to download only the best quality
audio:

```shell
$ ypb download -i 30s/now Mm_zVDDUeNA -- -f bestaudio -x
```

!!! tip "Picking a format"
    Want to see which formats are available? Run `yt-dlp -F` to list
    them (see [List available formats with
    yt-dlp](appendix/cookbook.md#list-available-formats-with-yt-dlp) for
    details). For the full syntax of the `-f` selector and more examples, see
    yt-dlp's [Format
    selection](https://github.com/yt-dlp/yt-dlp?tab=readme-ov-file#format-selection)
    docs.
    
## Where to go next

<div class="grid cards" markdown>

- **Command Line Interface**

    ---

    Full reference for commands, interval syntax, output naming, and more.
    
    :octicons-arrow-right-24: [Read the reference](reference/cli.md)

- **Cookbook**

    ---

    Practical recipes for common tasks, from listing formats to playing saved manifests.
    
    :octicons-arrow-right-24: [Browse recipes](appendix/cookbook.md)

</div>

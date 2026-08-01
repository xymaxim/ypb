# Download raw media segments

By default, ypb downloads YouTube live stream excerpts and merges them into a
single output file. To keep the raw media segments for your own post-processing
instead (including custom muxing and encoding, inspecting the embedded metadata,
etc.), use one of the two methods below.

## Prerequisites

- `ypb` installed locally, or running in a container
- `yt-dlp` installed (already included as a requirement)

## Download with yt-dlp

1. Append `-- --keep-fragments --ffmpeg-location /dev/null` to your download command:

   ```
   ypb download --interval 10s/now Mm_zVDDUeNA-g -- --keep-fragments --ffmpeg-location /dev/null
   ```

   - `--keep-fragments` keeps the individual `.part-FragN` files alongside each format's assembled file
   - `--ffmpeg-location /dev/null` points yt-dlp at a missing ffmpeg binary, so
     the final audio and video merge fails while fragment download and assembly
     (which don't need ffmpeg) still succeed.

   If you only need one stream, select it directly and drop `--ffmpeg-location`:
   
        ypb download --interval 10s/now Mm_zVDDUeNA-g -- --keep-fragments -f bestvideo

2. Run as usual. yt-dlp will report a merge warning, then download each format separately:

   ```
   (<<) Downloading and merging media...
   [generic] Extracting URL: http://localhost:9000/mpd
   [generic] mpd: Downloading webpage
   WARNING: [generic] Forcing generic information extractor
   [generic] mpd: Extracting information
   [info] mpd: Downloading 1 format(s): 313+140
   WARNING: You have requested merging of multiple formats but ffmpeg is not installed. The formats won't be merged
   [dashsegments] Total fragments: 2
   [download] Destination: MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f313.webm
   [download] 100% of    5.52MiB in 00:00:17 at 323.13KiB/s
   [dashsegments] Total fragments: 2
   [download] Destination: MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f140.m4a
   [download] 100% of  159.71KiB in 00:00:20 at 7.82KiB/s
   ```

3. Check your working directory:

   ```
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f140.m4a
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f140.m4a.part-Frag1
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f140.m4a.part-Frag2
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f313.webm
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f313.webm.part-Frag1
   MaunaKea-West-view-and-Meteor_MPokOMJvZ-g_20260731T085352+00_10s.f313.webm.part-Frag2
   ```

   The plain `.f140.m4a` / `.f313.webm` files are each format's assembled
   stream; `.part-FragN` are its raw segments.

## Download with dash-mpd-cli

As an alternative, dash-mpd-cli, a command-line application for downloading
media content from a MPEG-DASH manifest file, can save the raw media segments
and skip keeping the merged output entirely.

**Prerequisite:** [dash-mpd-cli](https://emarsden.github.io/dash-mpd-cli/)
installed.

1. Start a playback server for the video:

        ypb serve MPokOMJvZ-g


2. In another terminal, point `dash-mpd-cli` at it and save segments to a directory:

        dash-mpd-cli http://localhost:9000/mpd/10s--now --save-fragments output -o /dev/null

   Use `NUL` instead of `/dev/null` on Windows.

3. Check the output directory. Segments are split into `audio` and `video`
   subfolders:

   ```
   output
   ├── audio
   │   ├── 1974777
   │   └── 1974778
   └── video
       ├── 1974777
       └── 1974778
   ```

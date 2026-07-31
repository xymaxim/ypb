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

## Pass the output name to external programs

`ypb serve`'s JSON response includes `metadata.outputName`. It's the same name
`ypb download` would use for the same input internal. Reuse it when driving
another tool against the stream:

```shell
$ ypb serve STREAM
$ response=$(curl -s http://localhost:9000/mpd/10s--now)
$ output_name=$(echo "$response" | jq -r '.metadata.outputName')
```

For example, downloading a single muxed file with [gpac], piping the
manifest straight from the same response:

```shell
$ echo "$response" | jq -r '.mpd' \
  | gpac -i -:ext=mpd \
         -o "${output_name}.mp4":SID=#Representation=137,140
```

[gpac]: https://gpac.io/

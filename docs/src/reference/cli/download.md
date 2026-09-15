# download

```console exec="1" source="console"
$ ./build/ypb download --help
```

For rewind interval syntax, format selection, output filenames, and more, see
[Command Line Interface](cli.md).

## Cutting to input times

By default, `ypb download` fetches whole segments, so the saved file's start and
end are snapped to the nearest segment boundary rather than the exact times
requested. Pass `--cut`/`-c` to trim the output to the exact input start and end
times:

    ypb download -i <interval> --cut <stream>

Precise cuts require re-encoding the video stream, so `--cut` re-encodes video
using FFmpeg's default settings for the output container. The audio stream is
copied without re-encoding. This is equivalent to, with the extra options fixing
timing issues that can occur when copying the audio stream:

```sh

ffmpeg -ss <start> -i <input> -t <duration> -c:a copy \
       -avoid_negative_ts make_zero -shortest -fflags +genpts <output>
```

!!! warning "Re-encoding takes time"
    Re-encoding can take a long time, especially for longer intervals or
    high-resolution video.
    
To override the video encoding (or any other FFmpeg option), pass
`--cut-ffmpeg-options` with a string of ffmpeg options. It is appended after
the default options, so later values take precedence for conflicts:

```sh
ypb download -i 00:10:00-00:15:00 --cut \
  --cut-ffmpeg-options "-c:v libx264 -preset fast -crf 18" \
  https://example.com/stream
```

## Embedding metadata tags

By default, the `download` command adds metadata tags to a downloaded
excerpt file. Use the `--no-metadata` flag to disable it.

| Metadata tag      | Description                   | Example                                       |
|-------------------|-------------------------------|-----------------------------------------------|
| `Title`           | Video title                   | `Sample Title`                                |
| `Author`          | Video channel name            | `Sample Channel`                              |
| `Comment`         | YouTube video URL             | `https://www.youtube.com/watch?v=abcdefgh123` |
| `ActualStartTime` | Actual start time             | `2026-01-02T10:20:30.123Z`                    |
| `InputStartTime`  | Input start time              | `2026-01-02T10:20:30.000Z`                    |
| `ActualEndTime`   | Actual end time               | `2026-01-02T10:20:35.456Z`                    |
| `InputEndTime`    | Input end time                | `2026-01-02T10:20:35.000Z`                    |
| `StartSegment`    | Start segment sequence number | `1000`                                        |
| `EndSegment`      | End segment sequence number   | `1001`                                        |

!!! note "Input and actual times"
    The input time is the requested moment, while the actual time is snapped to
    the nearest matching segment boundary. See [Why does the actual time differ
    from the input
    time?](../../appendix/questions.md#why-does-the-actual-time-differ-from-the-input-time)

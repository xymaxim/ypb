# Command Line Interface

## Overview

```shell
Usage: ypb <command> [flags]

A playback for YouTube live streams

Flags:
  -h, --help       Show context-sensitive help.
  -v, --verbose    Show verbose output.

Commands:
  download --interval=STRING <stream> [<ytdlp-options> ...] [flags]
    Download stream excerpts

  serve <stream> [flags]
    Start playback server

  version [flags]
    Show version info and exit
```

## Commands 

### serve

```shell
Usage: ypb serve <stream> [flags]

Start playback server

Arguments:
  <stream>    YouTube video ID

Flags:
  -h, --help         Show context-sensitive help.
  -v, --verbose      Show verbose output.

  -p, --port=8080    Port to start playback on
```

### download 

```shell
Usage: ypb download --interval=STRING <stream> [<ytdlp-options> ...] [flags]

Download stream excerpts

Arguments:
  <stream>                 YouTube video ID
  [<ytdlp-options> ...]    Options to pass to yt-dlp (use after --)

Flags:
  -h, --help               Show context-sensitive help.
  -v, --verbose            Show verbose output.

  -i, --interval=STRING    Time or segment interval
  -p, --port=8080          Port to start playback on
```

#### Passing options to yt-dlp

Additional options can be passed directly to `yt-dlp` using the `--`
separator. Everything after `--` is forwarded to the underlying
`yt-dlp` process.

Behind the scences, `ypb` calls `yt-dlp` with a custom `-o/--output` template:
the only option that is overriden by default. This behavior can be changed as
follows:

    ypb download -i <interval> <stream> -- -o output.mp4

For a complete list of available options, see the [yt-dlp
documentation](https://github.com/yt-dlp/yt-dlp#usage-and-options).

> [!IMPORTANT]
> Not all `yt-dlp` options are compatible with `ypb`'s workflow. Incompatible
> options will not have effect. For example, [the network
> options](https://github.com/yt-dlp/yt-dlp?tab=readme-ov-file#network-options)
> are not supported.

## Specifying the rewind interval

The rewind interval is specified using the`-i/--interval` option. An
interval consists of start and end parts by either `/` or `--`:

```shell
$ ypb download -i <start>/<end>
$ ypb download -i <start>--<end>
```

### Absolute and relative moments

The interval parts refer to absolute or relative points (moments) in a
stream. Absolute moments independently indicate a specific point in time, while
relative moments are specified in relation to other moments.

Absolute moments can be further divided into direct and indirect types. Direct
moments correspond to exact stream media segments, while indirect moments
require locating a segment.

* **Absolute moments**
  - *Direct*: sequence numbers, the `now` keyword
  - *Indirect*: date and times, Unix timestamps, time arithmetic expressions

* **Relative moments**
  - Time durations

### Moment values

#### Date and time

* `<date-time> = <date>"T"<time>"±"<offset>`,

where `<date> = YYYY"-"MM"-"DD`, `<time> = "hh":"mm":"ss`,
and `<offset> = "±"hh":"mm`.

This format follows the extended ISO 8601 format or
[RFC3339](https://datatracker.ietf.org/doc/html/rfc3339.)

Here is an example of the complete representation with full time and partial
time offset in UTC:

    2026-01-02T10:20:30+00
    
The time component can be provided with reduced precision by omitting
lower-order components, which are assumed to be "00" (the date part must always
be complete):

```shell
# Complete date plus hours and minutes
2026-01-02T10:20+00

# Complete date plus hours only
2026-01-02T10+00
```

##### Zulu time 

Zulu time refers to UTC and is denoted by the letter "Z"
used as a suffix instead of a time offset:

    2026-01-02T10:20:30Z


##### Local time

To represent local time, omit the time offset. For example, if
you're in the UTC+02 time zone, the above example would be:

    2026-01-02T12:20:30

##### Time of today

To refer to a time of the current day, you can omit the date and time offset:

```shell
# Full time with time offset
10:20:30+00

# Full time in local time zone
10:20:30

# Hours and minutes only
10:20
```

#### Time duration

* `-i/--interval <start>/<duration>` or
* `-i/--interval <duration>/<end>`,

where `<duration> = dd"d"hh"h"mm"m"ss"s"`.

Sometimes it is more convenient to specify an interval using a duration. Duration strings
use single-letter designators for time components: days (`d`), hours
(`h`), minutes (`h`), and seconds (`s`).

The following examples represent the same interval from 10:30 to 12:00 (local time):

```shell
# Specified by start time and duration
--interval 10:30/1h30m ...

# Specified by duration and end time
--interval 1h30m/12:00 ...
```
  
#### Time arithmetic expression

* `<expression>`

where `<expression> = <operand> "±" <duration>` and `<operand>` is any absolute
moment. The expression also accepts the `now` keyword:
`<expression> = "now" "-" <duration>`.

Input moments can be represented as arithmetic expressions combining absolute
moments and durations. Such temporal arithmetic supports both addition and
subtraction. For example, the expression `10:30 - 30s` results in `10:00`. Use
the `now` keyword to refer to the current time.

Note that option values containing whitespace must be quoted.

```shell
# Subtraction between time and duration
--interval '2026-01-02T10:20:30 - 1d2h30m/30m' ...

# An excerpt centered around some specific time today
--interval '12:00 - 1m/12:00 + 5m' ...

# An excerpt spanning from yesterday 23:00 to today 01:00
--interval '23:00 - 1d/01:00' ...

# A 30-minute excerpt starting from one hour ago
--interval 'now - 1h/30m' ...
  ```

#### Sequence numbers

* `<sequence-number> = [0-9]+`

In addition to times, you can specify the sequence number (positive, starting
from 0) of an MPEG-DASH [media
segment](https://wiki.gpac.io/Howtos/dash/DASH-basics/#dash-basics-mpd-and-segments)
to reference a specific point in a live stream. Sequence numbers are typically
used when a segment has already been identified.

#### Keywords

##### 'Earliest' (*TODO*)

* `-i/--interval earliest/<end>`

To reference the earliest available moment, use the ``earliest``
keyword for the start part:

    --interval earliest/30m

This refers to either the beginning of the stream (the very first media segment)
or the earliest available segment if the stream has been running longer than the
available rewind window.

##### 'Now'

* `-i/--interval <start>/now`

To reference the current moment, use the now keyword for the end part:

    --interval 10:30/now

More precisely, this refers to the most recently available media segment, which
typically corresponds to the current moment. However, if a live stream has
stalled, new media segments are updated.

## Specifying the output filename

By default, downloaded files are saved in the current working directory with
names composed of the adjusted title, YouTube video ID, start time, and
duration:

```shell
$ ypb download -i 2026-01-02T10:20:30+00/30s abcdefgh123 && ls
Stream-title_abcdefgh123_20260102T102030+00_30s.mp4
```

To customize output names, use the yt-dlp's `-o/--output`
[option](https://github.com/ytdl-org/youtube-dl/#OUTPUT-TEMPLATE) by specifying
a full filename:

```shell
$ ypb download ... -- -o output/path.mp4 && ls output/*
output/path.mp4
```

Note that since `yt-dlp` downloads the MPEG-DASH manifest via the general
extractor rather than the YouTube extractor, YouTube-specific template
variables are not available.

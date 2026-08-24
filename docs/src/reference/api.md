# API

## Endpoints

### /

Serves the built-in web player (used by the `play` command).

#### Parameters

interval
:   The rewind interval or moment to play, given as the last path segment. Use
    the `--` separator for intervals (e.g., `10:00--12:00`).

tz
:   Timezone offset used for the playhead and output timestamps. Format
    `[+-]HH` or `[+-]HH:MM` (e.g., `+02`, `-05:30`). Defaults to UTC (`+00:00`).

latency, l
:   Correcting for streaming latency by locating the interval later by this many
    seconds (whole or fractional). See [Correcting for streaming
    latency](cli.md#correcting-for-streaming-latency) for details.

#### Usage examples

Play a 30-minute excerpt starting 30 minutes ago:

    http://localhost:9000/30m--now

Play from a moment and continue live:

    http://localhost:9000/12:00

Display timestamps in UTC+02:

    http://localhost:9000/12:00?tz=+02

### /info

Returns information about the YouTube live stream being served.

#### Response

```json
{
    "id": "0ujj4HexRpk",
    "title": "Stream title",
    "channelId": "UC6OWqjtFTsdtHAAuGWv1kPw",
    "channelTitle": "Channel name",
    "actualStartTime": "2026-01-02T10:20:30Z"
}
```

### /mpd/\{interval\}

Returns an MPEG-DASH manifest for the given interval. The manifest is *static*
when a bounded interval is provided, or *dynamic* when an open-ended interval
is provided.

#### Parameters

interval
:   The rewind interval to retrieve.

    !!! note
        See [Specifying the rewind interval](cli.md#specifying-the-rewind-interval)
        for all available interval format options. When using absolute timestamps,
        prefer the `Z` suffix for UTC (e.g., `2026-01-02T10:20:30Z`) over `+00:00`,
        since `+` must be percent-encoded as `%2B`. In general, ensure the path
        parameter is properly URL-encoded: use `--` as the interval separator
        instead of `/` and avoid unencoded whitespace.

latency, l
:   Correcting for streaming latency by locating the interval later by this many
    seconds (whole or fractional). See [Correcting for streaming
    latency](cli.md#correcting-for-streaming-latency) for details.

#### Usage examples


Rewind a 30-minute excerpt starting at 12:00 (static):

    $ curl localhost:9000/mpd/12:00--30m

Playback starting from 12:00, continuing live (dynamic):

    curl localhost:9000/mpd/12:00

Same, corrected for 10 seconds of streaming latency:

    curl localhost:9000/mpd/12:00?latency=10
    
#### Response

By default, returns the raw MPEG-DASH manifest as `application/dash+xml`. To
receive a JSON representation including the raw manifest and metadata, set
the `Accept: application/json` header.

The JSON response has the following structure:

```json
{
    "metadata": {
        "videoTitle": "Stream title",
        "videoUrl": "https://www.youtube.com/live/...",
        "outputName": "Stream-title_abcdefgh123_20260102T102030+00_30m",
        "startActualTime": "2026-01-02T10:00:02Z",
        "startTargetTime": "2026-01-02T10:00:00Z",
        "endActualTime": "2026-01-02T10:30:03Z",
        "endTargetTime": "2026-01-02T10:30:00Z",
    },
    "mpd": "<?xml version=\"1.0\" ...>"
}
```

For dynamic manifests, `outputName`, `endActualTime`, and `endTargetTime` are
omitted.

### /segments/itag/\{itag\}/sq/\{sq\}

Serves a media segment indentified by itag and sequence number.

#### Parameters

itag
: The segment itag value.

sq
: The segment sequence number.

#### Response

The bytes of the requested media segment.

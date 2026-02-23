# API

## Endpoints

### /mpd/\{interval\}

Returns an MPEG-DASH manifest for the given interval. The manifest is *static*
when a bounded interval is provided, or *dynamic* when an open-ended interval
is provided.

#### Parameters

interval
: The rewind interval to retrieve.

  > [!NOTE]
  > See [Specifying the rewind interval](cli.md#specifying-the-rewind-interval)
  > for all available interval format options. When using absolute timestamps,
  > prefer the `Z` suffix for UTC (e.g., `2026-01-02T10:20:30Z`) over `+00:00`,
  > since `+` must be percent-encoded as `%2B`. In general, ensure the path
  > parameter is properly URL-encoded: use `--` as the interval separator
  > instead of `/` and avoid unencoded whitespace.

#### Usage examples

Rewind a 30-minute excerpt from one day ago (static):

    $ curl localhost:8080/mpd/now-1d--30m


Playback starting from ten minutes ago, continuing live (dynamic):

    curl localhost:8080/mpd/now-10m


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
        "startActualTime": "2026-01-02T10:00:02Z",
        "startTargetTime": "2026-01-02T10:00:00Z",
        "endActualTime": "2026-01-02T10:30:03Z",
        "endTargetTime": "2026-01-02T10:30:00Z",
    },
    "mpd": "<?xml version=\"1.0\" ...>"
}
```

For dynamic manifests, `endActualTime` and `endTargetTime` are omitted.

### /segments/itag/\{itag\}/sq/\{sq\}

Serves a media segment indentified by itag and sequence number.

#### Parameters

itag
: The segment itag value.

sq
: The segment sequence number.

#### Response

The bytes of the requested media segment.

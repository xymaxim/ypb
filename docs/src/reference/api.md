# API

## Endpoints

### /rewind/\{interval\}

Returns a static MPEG-DASH manifest representing media for the rewind interval.

*Parameters*

interval
: The rewind interval to retrieve.

  > [!NOTE]
  > See [Specifying the rewind interval](cli.md#specifying-the-rewind-interval) for
  > available format options. Note that the path parameter should be properly
  > URL-encoded: use `--` instead of `/`, and avoid whitespaces or use percent
  > encoding.
  
*Usage examples*

Rewind a 30-minute excerpt from one day:

    $ curl localhost:8080/rewind/now-1d--30m

*Response*

The raw content of the composed MPEG-DASH manifest.

### /videoplayback/itag/\{itag\}/sq/\{sq\}

Serves a media segment indentified by itag and sequence number.

*Parameters*

itag
: The segment itag value.

sq
: The segment sequence number.

*Response*

The bytes of the requested media segment.

# Pre-built binaries

## Prerequisites

While ypb itself is lightweight, it relies on yt-dlp:

* [yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/Installation): For video info
  extraction and downloading. *Nightly builds are recommended. If you use
  binaries, update with: `yt-dlp --update-to nightly`*.

* [FFmpeg](https://ffmpeg.org/) (*optional*): For muxing downloads with `ypb
  download`

### Additional dependencies

The following dependencies are optional, but strongly recommended in practice:

* [External JavaScript runtime](https://github.com/yt-dlp/yt-dlp/issues/15012):
  Required for full YouTube support

* Proof-of-Origin (PO) token [provider
  plugin](https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide): Required to
  avoid HTTP 403 errors

## Install from binaries

Pre-built binaries for different platforms are available on the GitHub [latest
release](https://github.com/xymaxim/ypb/releases/latest) page:

*Latest release {{git.short_tag}}*

<div class="grid cards three-cols" markdown>

-   :material-linux:{ .lg } **Linux**

    [amd64][ypb-{{ git.short_tag }}-linux-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-linux-arm64.zip]

-   :material-apple:{ .lg } **macOS**

    [amd64][ypb-{{ git.short_tag }}-darwin-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-darwin-arm64.zip]

-   :material-microsoft-windows:{ .lg } **Windows**

    [amd64][ypb-{{ git.short_tag }}-windows-amd64.zip] · [arm64][ypb-{{ git.short_tag }}-windows-arm64.zip]

</div>

[ypb-{{ git.short_tag }}-linux-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-linux-amd64.zip
[ypb-{{ git.short_tag }}-linux-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-linux-arm64.zip
[ypb-{{ git.short_tag }}-darwin-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-darwin-amd64.zip
[ypb-{{ git.short_tag }}-darwin-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-darwin-arm64.zip
[ypb-{{ git.short_tag }}-windows-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-windows-amd64.zip
[ypb-{{ git.short_tag }}-windows-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ git.short_tag }}/ypb-{{ git.short_tag }}-windows-arm64.zip

### Linux/macOS

Download the binary for your operating system using the links above, and place
it to a directory that is in your `PATH`. Make the binary executable with `chmod
+x`.

Once installed, verify the installation: 

```shell
ypb version
``` 

### Windows

Download the binary using the links above, and extract it to a permanent
location such as `C:\Program Files\ypb\`. Add this directory to your system
`PATH` via Environment Variables settings to make the binary accessible from any
location in PowerShell.

## Setup

While optional, the setup steps below are strongly recommended.

### Update yt-dlp to nightly

YouTube changes frequently, and the stable yt-dlp release can lag behind.
Switch to the nightly build, which is updated daily:

```shell
yt-dlp --update-to nightly
```

Check the result with `yt-dlp --version`.

### Configure yt-dlp

Ypb relies on yt-dlp for specific tasks like fetching video information and
downloading. Because of that, your [yt-dlp configuration
file](https://github.com/yt-dlp/yt-dlp#configuration) also apply to ypb: you can
set the formats, cookies (see below), and other options there as usual.

### Sign in with cookies

Some streams respond with a "Sign in to confirm you're not a bot" error unless
you provide cookies. Export your YouTube cookies from a logged-in browser (see
[yt-dlp's instructions](https://github.com/yt-dlp/yt-dlp/wiki/Extractors#exporting-youtube-cookies)),
then add a line to your [yt-dlp configuration
file](https://github.com/yt-dlp/yt-dlp#configuration):

```text
--cookies cookies.txt
```

or read them straight from the browser:

```text
--cookies-from-browser chrome
```

### Install a JavaScript runtime

If yt-dlp cannot find a JavaScript runtime, you will see this warning:

```text
WARNING: [youtube] No supported JavaScript runtime could be found. Only deno is
enabled by default; to use another runtime add --js-runtimes RUNTIME[:PATH] to
your command/config. YouTube extraction without a JS runtime has been
deprecated, and some formats may be missing. See
https://github.com/yt-dlp/yt-dlp/wiki/EJS for details on installing one
```

Install [Deno](https://deno.com/) (version 2.0 or later). If it is not detected
automatically, point `yt-dlp` at it:

```text
--js-runtimes deno:/path/to/deno
```

See the [EJS wiki](https://github.com/yt-dlp/yt-dlp/wiki/EJS) for other
supported runtimes.

### Avoid HTTP 403 errors with a PO token provider

During downloads you may see transient HTTP 403 errors, caused by YouTube's bot
verification. yt-dlp works around them with a Proof-of-Origin (PO) token
provider plugin:

1. Install the
   [bgutil-ytdlp-pot-provider](https://github.com/Brainicism/bgutil-ytdlp-pot-provider)
   plugin.
2. Run the provider's HTTP server, which listens on `http://127.0.0.1:4416` by
   default.

With the default address, no further configuration is needed. If you run the
server elsewhere, point yt-dlp at it:

```text
--extractor-args "youtubepot-bgutilhttp:base_url=http://127.0.0.1:8080"
```

See yt-dlp's [PO Token Guide](https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide)
for details.

## Install from source

If you have Go installed, you can build `ypb` from source:

``` shell
go install github.com/xymaxim/ypb@latest
```

This is not recommended unless you need a development version.

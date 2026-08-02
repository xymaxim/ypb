# Pre-built binaries

## Prerequisites

While `ypb` itself is lightweight, it relies on `yt-dlp`:

* [yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/Installation): For video info
  extraction and downloading. Nightly builds are recommended. If you use
  binaries, update with: `yt-dlp --update-to nightly`.

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
release](https://github.com/xymaxim/ypb/releases/latest) page.

|       | Linux                                       | macOS                                        | Windows                                       |
|-------|---------------------------------------------|----------------------------------------------|-----------------------------------------------|
| AMD64 | [ypb-{{ git.short_tag }}-linux-amd64.zip] | [ypb-{{ git.short_tag }}-darwin-amd64.zip] | [ypb-{{ git.short_tag }}-windows-amd64.zip] |
| ARM64 | [ypb-{{ git.short_tag }}-linux-arm64.zip] | [ypb-{{ git.short_tag }}-darwin-arm64.zip] | [ypb-{{ git.short_tag }}-windows-arm64.zip] |

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

## Install from source

If you have Go installed, you can build `ypb` from source:

``` shell
go install github.com/xymaxim/ypb@latest
```

Note: this is not recommended unless you need a development version.

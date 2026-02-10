# Pre-built binaries

## Prerequisites

Ensure the following [requirements](install.md#requirements) are installed based
on your intended usage:

* `ypb serve`: yt-dlp with additional dependencies (to avoid 403 HTTP errors)
* `ypb download`: All of the above plus FFmpeg (for muxing during downloads)

## Install from binaries

Pre-built binaries for different platforms are available on the GitHub [latest
release](https://github.com/xymaxim/ypb/releases/latest) page.

|       | Linux                            | macOS                             | Windows                            |
|-------|----------------------------------|-----------------------------------|------------------------------------|
| AMD64 | [ypb-v2026.2.10-linux-amd64.zip] | [ypb-v2026.2.10-darwin-amd64.zip] | [ypb-v2026.2.10-windows-amd64.zip] |
| ARM64 | [ypb-v2026.2.10-linux-arm64.zip] | [ypb-v2026.2.10-darwin-arm64.zip] | [ypb-v2026.2.10-windows-arm64.zip] |

[ypb-v2026.2.10-linux-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-linux-amd64.zip
[ypb-v2026.2.10-linux-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-linux-arm64.zip
[ypb-v2026.2.10-darwin-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-darwin-amd64.zip
[ypb-v2026.2.10-darwin-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-darwin-arm64.zip
[ypb-v2026.2.10-windows-amd64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-windows-amd64.zip
[ypb-v2026.2.10-windows-arm64.zip]: https://github.com/xymaxim/ypb/releases/download/{{ release_version }}/ypb-{{ release_version }}-windows-arm64.zip

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

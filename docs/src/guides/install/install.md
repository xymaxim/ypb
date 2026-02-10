# Install

Ypb works on Linux, macOS, and Windows. 

There are two installation methods available:

1. [**Pre-built binaries**](binaries.md): Install platform-specific binaries
   and along with additional dependencies
2. [**Container image**](container.md): Run in a container with all dependencies
   bundled

## Choosing installation method

The choice depends on your current setup and usage:

| Feature      | Pre-built binaries                                                        | Container image                                       |
|--------------|---------------------------------------------------------------------------|-------------------------------------------------------|
| Setup        | You already have yt-dlp and FFmpeg installed with additional dependencies | You want a self-contained setup with all dependencies |
| Installation | Manual installation of binaries and dependencies                          | Requires Podman or Docker                             |
| Updates      | Manual updating of all dependencies                                       | Updating container image                              |

## Requirements

While `ypb` itself is lightweight, it relies on `yt-dlp`:

* [yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/Installation): For video info
  extraction and downloading. Nightly builds are recommended. If you use
  binaries, update with: `yt-dlp --update-to nightly`

* [FFmpeg](https://ffmpeg.org/) (*optional*): For muxing downloads with `ypb
  download`.

### Additional dependencies

The following dependencies are optional but strongly recommended:

* [External JavaScript runtime](https://github.com/yt-dlp/yt-dlp/issues/15012):
  Required for full YouTube support

* Proof-of-Origin (PO) token [provider
  plugin](https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide): Required to
  avoid HTTP 403 errors

# Install

Ypb works on Linux, macOS, and Windows. 

There are two installation methods available:

1. [**Pre-built binaries**](binaries.md): Install platform-specific binaries
   along with additional dependencies
2. [**Container (Compose)**](container.md): Run in a container with all dependencies
   bundled

## Choosing installation method

The choice depends on your current setup and usage:

| Feature      | Pre-built binaries                                                        | Container image                                       |
|--------------|---------------------------------------------------------------------------|-------------------------------------------------------|
| Setup        | You already have yt-dlp and FFmpeg installed with additional dependencies | You want a self-contained setup with all dependencies |
| Installation | Manual installation of binaries and dependencies                          | Requires Podman or Docker                             |
| Updates      | Manual updating of all dependencies                                       | Updating container image                              |

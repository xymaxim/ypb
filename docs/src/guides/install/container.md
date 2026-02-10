# Container image

## Prerequisites

No additional dependencies required: the container image includes all necessary
components (yt-dlp, FFmpeg, and additional dependencies).

You'll need either [Podman](https://podman.io/getting-started/installation)
(recommended) or [Docker](https://docs.docker.com/get-docker/).

### Initial setup

On macOS and Windows, Podman requires a virtual machine. Initialize and start it
once:

```shell
podman machine init
podman machine start
```

The machine will automatically start on reboots. You can verify it is running:

```shell
podman machine list
```

## Pull the image

Pull the latest container image from GitHub Container Registry:

```shell
podman pull ghcr.io/xymaxim/ypb
```

## Usage

### Basic commands

Run `ypb` commands directly with the container:

```shell
podman run --rm ghcr.io/xymaxim/ypb version
```

### Recommended aliases

For easier usage, add these aliases to your shell configuration file:

```shell
# General commands like ypb version
alias ypb='podman run --rm ghcr.io/xymaxim/ypb'

# Downloads videos to current directory (mounts volume)
alias ypb-download='podman run --rm -v .:/content ghcr.io/xymaxim/ypb download'

# Starts server accessible at `http://localhost:8080` (exposes port)
alias ypb-serve='podman run --rm -p 8080:8080 ghcr.io/xymaxim/ypb serve'
```

> [!IMPORTANT]
> On SELinux-enabled systems add `:Z` to the volume mount to avoid permission
> errors.

### Manual usage without aliases

If you prefer not to use aliases or need custom configurations:

**Download videos to current directory:**

```shell
podman run --rm -v .:/content ghcr.io/xymaxim/ypb download 
```

**Start server on port 8080:**

```shell
podman run --rm -p 8080:8080 ghcr.io/xymaxim/ypb serve
```

**Use custom port (e.g., 3000):**

```shell
podman run --rm -p 3000:8080 ghcr.io/xymaxim/ypb serve
```

## Update the image

To update `ypb` and all dependecies to the latest version:

```shell
podman pull ghcr.io/xymaxim/ypb
```

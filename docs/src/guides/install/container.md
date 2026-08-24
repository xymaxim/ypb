# Container (Compose)

Running ypb with containers is the recommended way to get started.

!!! note
    This guide uses Podman, but Docker works too. Commands are mostly the
    same with `docker` in place of `podman`, though some steps (like
    `podman machine` and `podman artifact`) don't have a direct Docker
    equivalent.

The app runs as two containers managed by [Compose](https://compose-spec.io/):

- **Ypb** ([ghcr.io/xymaxim/ypb](https://ghcr.io/xymaxim/ypb)) — the main app, with yt-dlp and ffmpeg inside
- **PO token provider** ([brainicism/bgutil-ytdlp-pot-provider](https://hub.docker.com/r/brainicism/bgutil-ytdlp-pot-provider)) — handles YouTube's bot verification in the background

## Prerequisites

- [Podman](https://podman.io/getting-started/installation) with [Compose](https://podman-desktop.io/docs/compose)

- YouTube cookies [exported](https://github.com/yt-dlp/yt-dlp/wiki/Extractors#exporting-youtube-cookies) from your browser

### macOS and Windows

On macOS and Windows, Podman requires a virtual machine. Initialize and start
it once:

```shell
podman machine init
podman machine start
```

The machine starts automatically on subsequent reboots.

## Set up

1. Pull the compose file and extract it to a local directory:

   ```shell
   podman artifact pull ghcr.io/xymaxim/ypb-compose
   podman artifact extract ghcr.io/xymaxim/ypb-compose ~/ypb-app
   cd ~/ypb-app
   ```

   This gives you `compose.yaml` and `.env.template` files with defaults,
   containing configuration variables — see [Configuration](#configuration)
   below for what's available.

2. Copy `.env.template` to `.env` and edit that copy:

   ```shell
   cp .env.template .env
   ```

   `.env` is yours to customize and won't be overwritten by future updates.

### Configure yt-dlp

Ypb relies on yt-dlp for specific tasks, like fetching video info and
downloading, and uses its [configuration
file](https://github.com/yt-dlp/yt-dlp#configuration): you can set the formats
to download, cookies, and other options there. In the container, edit the config
file (`config` or `config.txt`) inside your `YPB_YTDLP_CONFIG_DIR` directory —
see [Configuration](#configuration).

### Set up cookies (recommended)

YouTube may respond with a "Sign in to confirm you're not a bot" error
without cookies, so setting them up is recommended. To avoid this:

1. Export cookies from your browser into a `cookies.txt` file.
2. In `.env`, set `YPB_YTDLP_CONFIG_DIR` to the directory where you want
   to store yt-dlp related config files.
3. Place `cookies.txt` inside that directory.
4. Reference it from your yt-dlp config file (`config`, `config.txt`):

        --cookies /path/to/cookies.txt

## Usage

Run as many download commands as needed:

```shell
podman compose run --rm ypb download -i 30s/now abcdefgh123
```

To start a playback server, use `serve`:

```shell
podman compose run --rm ypb serve abcdefgh123
```

This listens on port 9000.

When done, shut down the PO token provider sidecar:

```shell
podman compose down
```

## Configuration

The `.env.template` file (copied to `.env` during [setup](#set-up)) holds
environment variables you can set:

### YPB_MEDIA_DIR

Where output media files are saved on your machine. The directory is
created automatically if it doesn't already exist.

### YPB_YTDLP_CONFIG_DIR

By default, ypb uses its own built-in yt-dlp configuration. Mounting your
own config directory here lets you add your own settings on top — see
[Set up cookies](#set-up-cookies-recommended) above for an example.

## Update the app

To update the container images:

```shell
podman compose pull
```

To pick up changes to `compose.yaml` or `.env.template`, re-run the extract
step. This leaves your `.env` untouched:

```shell
podman artifact pull ghcr.io/xymaxim/ypb-compose
podman artifact extract ghcr.io/xymaxim/ypb-compose .
```

# Container (Compose)

Running ypb with containers is the recommended way to get started.

> [!NOTE]
> This guide uses Podman, but Docker works too. Commands are mostly the
> same with `docker` in place of `podman`, though some steps (like
> `podman machine` and `podman artifact`) don't have a direct Docker
> equivalent.

The app runs as two containers managed by [Compose](https://compose-spec.io/):

- **Ypb** ([ghcr.io/xymaxim/ypb](https://ghcr.io/xymaxim/ypb)) — the main app, with yt-dlp and ffmpeg inside
- **PO token provider** ([brainicism/bgutil-ytdlp-pot-provider](https://hub.docker.com/r/brainicism/bgutil-ytdlp-pot-provider)) — handles YouTube's bot verification in the background

## Prerequisites

[Podman](https://podman.io/getting-started/installation) with [Compose](https://podman-desktop.io/docs/compose).

### macOS and Windows

On macOS and Windows, Podman requires a virtual machine. Initialize and start
it once:

```shell
podman machine init
podman machine start
```

The machine starts automatically on subsequent reboots.

## Set up

Pull the compose file and extract it to a local directory:

```shell
podman artifact pull ghcr.io/xymaxim/ypb-compose
podman artifact extract ghcr.io/xymaxim/ypb-compose ~/ypb-app
cd ~/ypb-app
```

This gives you `compose.yaml` and a `.env` file with commented-out defaults.
Edit `.env` if you want to override where files are saved or use your own yt-dlp
config. Otherwise, the defaults work out of the box.

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

The `.env` file holds environment variables you can set:

### YPB_MEDIA_DIR

Where output media files are saved on your machine. The directory is
created automatically if it doesn't already exist.

### YPB_YTDLP_CONFIG_DIR

By default, `ypb` uses its own built-in yt-dlp configuration. Mounting your
own config directory here lets you add your own settings on top.

For example, to use cookies, export them from your browser into a `cookies.txt`
file, put it inside your `YPB_YTDLP_CONFIG_DIR`, then reference it from your yt-dlp config
file (`config`, `config.txt`):

```
--cookies /path/to/cookies.txt
```

## Update the app

```shell
podman compose pull
```

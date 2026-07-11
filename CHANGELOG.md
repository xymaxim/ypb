# Changelog

The format of this changelog is based on [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Calendar
Versioning](https://calver.org).

## [2026.7.12](https://github.com/xymaxim/ypb/releases/tag/v2026.7.12)

### Fixed

- Update dump format parsing and base URL handling (#7)

### Added

- Accept passing yt-dlp options in most commands

## [2026.6.15](https://github.com/xymaxim/ypb/releases/tag/v2026.6.15)

### Changed

- Supress console windows on Windows

## [2026.6.11](https://github.com/xymaxim/ypb/releases/tag/v2026.6.11)

### Added

- Add playback and stream public packages
- Add parser for fractional seconds
- Add `/info` endpoint returning basic info about YouTube live stream

### Fixed

- Set MPD@availabilityStartTime to now for dynamic DASH manifests
### Changed

- Use yt-dlp's --write-info-json to dump info JSON
- Change license from MIT to GPLv3

## [2026.2.24](https://github.com/xymaxim/ypb/releases/tag/v2026.2.24)

### Added

- Accept open-ended interval in `/mpd/` endpoint to compose dynamic MPD

### Fixed

- Pin start-up time as `now` in strict mode (`download` and `capture` commands) (#5)

### Changed

- Rename `/rewind/` endpoint to `/mpd/`
- Rename `/videoplayback/` endpoint to `/segments/`
- Normalize MPDs to be playbable in Shaka player
- Avoid downloading same segment in capture timelapse with small intervals
- Switch to custom text progress bar in capture timelapse

## [2026.2.18](https://github.com/xymaxim/ypb/releases/tag/v2026.2.18)

### Changed

- Retry on connection errors ("connection reset by peer", "connection timed out", etc)
- Print standard output and error from external commands directly (#2)

## [2026.2.16](https://github.com/xymaxim/ypb/releases/tag/v2026.2.16)

### Added

- New `capture frame` command to capture a single frame
- New `capture timelapse` command to capture multiple frames

### Fixed

- Incorrect 12-hour format (changed to 24-hour) in output filenames

### Changed

- Rework CommandRunner to accept functional options

## [2026.2.12](https://github.com/xymaxim/ypb/releases/tag/v2026.2.12)

### Fixed

- Incorrect parsing of 'now' in interval expressions (#1)

## [2026.2.10](https://github.com/xymaxim/ypb/releases/tag/v2026.2.10)

First release.

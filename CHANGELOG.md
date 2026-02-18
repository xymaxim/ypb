# Changelog

The format of this changelog is based on [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Calendar
Versioning](https://calver.org).

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

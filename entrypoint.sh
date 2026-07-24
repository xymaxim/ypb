#!/bin/sh
set -e
/usr/local/bin/yt-dlp --update-to nightly
exec /usr/local/bin/ypb "$@"

#!/bin/sh
/usr/local/bin/yt-dlp --update-to nightly
exec /usr/local/bin/ypb "$@"

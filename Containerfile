# Stage 1. Build ypb
FROM golang:1.25-alpine AS builder

RUN apk update && apk add --no-cache make git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# Stage 2. Create the final image
FROM alpine:latest

LABEL org.opencontainers.image.title="Ypb"
LABEL org.opencontainers.image.description="A playback for YouTube live streams."
LABEL org.opencontainers.image.source="https://github.com/xymaxim/ypb"
LABEL org.opencontainers.image.licenses="GPL-3.0-or-later"

RUN apk update && apk add --no-cache git deno ffmpeg
RUN apk add --no-cache ca-certificates && update-ca-certificates

RUN wget -q -O /usr/local/bin/yt-dlp \
    "https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/yt-dlp_musllinux" && \
    chmod 755 /usr/local/bin/yt-dlp

COPY --from=builder --chmod=755 /build/ypb /usr/local/bin
RUN /usr/local/bin/ypb version

COPY --chmod=755 entrypoint.sh /usr/local/bin/entrypoint.sh

WORKDIR /content

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

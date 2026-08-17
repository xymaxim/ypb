# Generating fixture for mock playback

`cmd/mockplay`, `internal/mockserver`, and `ypb.NewMockStream` serve a fake live
stream from locally generated media instead of hitting YouTube. The
[`ghcr.io/xymaxim/ypb-mock`](https://github.com/xymaxim/ypb/pkgs/container/ypb-mock)
image builds that fixture data using ffmpeg.

Run the following to generate a fixture:

```sh
mkdir -p playback/testdata/fixture
podman run --rm \
  -v "$(pwd)/playback/testdata/fixture:/output:Z" \
  ghcr.io/xymaxim/ypb-mock:latest
```

This produces an hour of content per itag in 5-second segments, plus
`info.json`:

    playback/testdata/fixture/
      info.json
      segments/134/1.mp4 ... 720.mp4
      segments/135/1.mp4 ... 720.mp4
      segments/140/1.mp4 ... 720.mp4

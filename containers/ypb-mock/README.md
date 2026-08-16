# Generating fixture for mock playback

`cmd/mockplay`, `internal/mockserver`, and `ypb.NewMockStream` serve a fake live
stream from locally generated media instead of hitting YouTube. The
[`ghcr.io/xymaxim/ypb-mock`](https://github.com/xymaxim/ypb/pkgs/container/ypb-mock)
image builds that fixture data using ffmpeg.

Run the following to generate a fixture:

```sh
mkdir -p internal/playback/testdata/fixture
podman run --rm \
  -v "$(pwd)/internal/playback/testdata/fixture:/work:Z" \
  ghcr.io/xymaxim/ypb-mock:latest
```

This produces an hour of content per itag in 5-second segments, plus
`info.json`:

    internal/playback/testdata/fixture/
      info.json
      segments/itag/134/1.mp4 ... 720.mp4
      segments/itag/135/1.mp4 ... 720.mp4
      segments/itag/140/1.mp4 ... 720.mp4

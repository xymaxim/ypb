package main

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/commands"
)

const (
	downloadCommand = "download"
	serveCommand    = "serve"
	hostFlag        = "--host"
	nowFlag         = "--now"
	latencyFlag     = "--latency"
	intervalFlag    = "-i"
	validInterval   = "1m/now"
	invalidInterval = "1m/2m"
	streamID        = "stream-id"
)

func TestDownloadFlags(t *testing.T) {
	t.Parallel()

	var cli CLI
	parser, err := kong.New(&cli)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		downloadCommand,
		hostFlag, "localhost",
		"-p", "8080",
		nowFlag, "2026-01-02T10:20:30+00",
		latencyFlag, "1.5",
		intervalFlag, validInterval,
		streamID,
	})
	require.NoError(t, err)

	assert.Equal(t, commands.Host("localhost"), cli.Download.Host)
	assert.Equal(t, commands.Port(8080), cli.Download.Port)
	assert.Equal(t, commands.PinnedTime("2026-01-02T10:20:30+00"), cli.Download.Now)
	assert.InDelta(t, 1.5, float64(cli.Download.Latency), 0)
	assert.Equal(t, validInterval, cli.Download.Interval)
	assert.Equal(t, streamID, cli.Download.Stream)
}

func TestFlagDefaults(t *testing.T) {
	t.Parallel()

	var cli CLI
	parser, err := kong.New(&cli)
	require.NoError(t, err)

	_, err = parser.Parse([]string{downloadCommand, intervalFlag, validInterval, streamID})
	require.NoError(t, err)

	assert.Equal(t, commands.Host(""), cli.Download.Host)
	assert.Equal(t, commands.Port(9000), cli.Download.Port)
	assert.Equal(t, commands.PinnedTime(""), cli.Download.Now)
	assert.InDelta(t, 0.0, float64(cli.Download.Latency), 0)
}

func TestServeFlags(t *testing.T) {
	t.Parallel()

	t.Run("explicit", func(t *testing.T) {
		t.Parallel()

		var cli CLI
		parser, err := kong.New(&cli)
		require.NoError(t, err)

		_, err = parser.Parse([]string{
			serveCommand,
			hostFlag, "[::1]",
			"-p", "8080",
			"--ui",
			streamID,
		})
		require.NoError(t, err)

		assert.Equal(t, commands.Host("::1"), cli.Serve.Host)
		assert.Equal(t, commands.Port(8080), cli.Serve.Port)
		assert.True(t, cli.Serve.UI)
		assert.Equal(t, streamID, cli.Serve.Stream)
	})

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		var cli CLI
		parser, err := kong.New(&cli)
		require.NoError(t, err)

		_, err = parser.Parse([]string{serveCommand, streamID})
		require.NoError(t, err)

		assert.Equal(t, commands.Host(""), cli.Serve.Host)
		assert.Equal(t, commands.Port(9000), cli.Serve.Port)
		assert.False(t, cli.Serve.UI)
	})

	t.Run("rejects now", func(t *testing.T) {
		t.Parallel()

		var cli CLI
		parser, err := kong.New(&cli)
		require.NoError(t, err)

		_, err = parser.Parse([]string{serveCommand, nowFlag, "2026-01-02", streamID})
		require.ErrorContains(t, err, nowFlag)
	})
}

func TestInvalidTypedFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			"host",
			[]string{serveCommand, hostFlag, "::1::", streamID},
			"not a valid address",
		},
		{
			"port",
			[]string{serveCommand, "-p", "70000", streamID},
			"between 0 and 65535",
		},
		{
			"now",
			[]string{
				downloadCommand,
				intervalFlag,
				invalidInterval,
				nowFlag,
				"1h",
				streamID,
			},
			"must be a time",
		},
		{
			"latency negative",
			[]string{
				downloadCommand,
				intervalFlag,
				invalidInterval,
				"--latency=-1",
				streamID,
			},
			"must not be negative",
		},
		{
			"latency not a number",
			[]string{
				downloadCommand,
				intervalFlag, invalidInterval,
				latencyFlag, "100s",
				streamID,
			},
			"invalid seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cli CLI
			parser, err := kong.New(&cli)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestEnvFlags is not parallel because it uses t.Setenv.
func TestEnvFlags(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		value   string
		args    []string
		wantErr string
	}{
		{
			"host",
			"YPB_HOST",
			"::1::",
			[]string{serveCommand, streamID},
			"not a valid address",
		},
		{
			"port",
			"YPB_PORT",
			"70000",
			[]string{serveCommand, streamID},
			"between 0 and 65535",
		},
		{
			"now",
			"YPB_NOW",
			"1h",
			[]string{downloadCommand, intervalFlag, invalidInterval, streamID},
			"must be a time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.env, tt.value)

			var cli CLI
			parser, err := kong.New(&cli)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

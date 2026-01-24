package playback

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func NewClient(pb *Playback) *http.Client {
	client := &retryablehttp.Client{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		RetryMax:   3,
		CheckRetry: makeRetryPolicy(pb),
	}
	return client.StandardClient()
}

func makeRetryPolicy(pb *Playback) retryablehttp.CheckRetry {
	return func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil {
			msg := "got nil response"
			slog.Error(msg, "err", err)
			return false, fmt.Errorf(msg+": %w", err)
		}

		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusServiceUnavailable:
			slog.Warn("got recoverable HTTP error, retrying", "code", resp.StatusCode)
			if resp.StatusCode == http.StatusForbidden {
				if err := pb.RefreshBaseURLs(); err != nil {
					return false, fmt.Errorf(
						"refreshing base URLs before retry: %w",
						err,
					)
				}
			}
			return true, nil
		default:
			return false, nil
		}
	}
}

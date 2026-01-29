package playback

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

func NewClient(pb Playbacker) *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.CheckRetry = makeRetryPolicy(pb)
	return client
}

func makeRetryPolicy(pb Playbacker) retryablehttp.CheckRetry {
	return func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil {
			return false, fmt.Errorf("got nil response: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusServiceUnavailable:
			slog.Warn("got recoverable HTTP error, retrying", "status", resp.StatusCode)
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

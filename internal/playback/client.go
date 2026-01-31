package playback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

func NewClient(pb Playbacker) *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.CheckRetry = makeRetryPolicy(pb)
	client.Logger = slog.Default()
	return client
}

func makeRetryPolicy(pb Playbacker) retryablehttp.CheckRetry {
	return func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil {
			return false, errors.New("got nil response")
		}

		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusServiceUnavailable:
			slog.Warn("got transient HTTP error, retrying", "status", resp.StatusCode)
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

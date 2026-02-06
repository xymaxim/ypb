package playback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/xymaxim/ypb/internal/urlutil"
)

func NewClient(pb Playbacker) *retryablehttp.Client {
	client := retryablehttp.NewClient()

	client.HTTPClient.Timeout = time.Minute

	client.Backoff = func(
		minimum, maximum time.Duration,
		attempt int,
		resp *http.Response,
	) time.Duration {
		wait := retryablehttp.DefaultBackoff(minimum, maximum, attempt, resp)
		slog.Warn(
			fmt.Sprintf(
				"retrying request in %v seconds, attempt %d of %d",
				wait.Seconds(), attempt+1, client.RetryMax,
			),
		)
		return wait
	}

	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil {
			return false, errors.New("got nil response")
		}

		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusServiceUnavailable:
			slog.Warn(
				"got transient HTTP error, retrying",
				"status", resp.StatusCode,
				"method", resp.Request.Method,
				"url", resp.Request.URL,
			)
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

	client.PrepareRetry = func(req *http.Request) error {
		// Extract the itag parameter
		itag := urlutil.ExtractParameter(req.URL.Path, "itag")
		if itag == "" {
			return nil
		}

		baseURLString := pb.BaseURLs()[itag]
		if baseURLString == "" {
			return fmt.Errorf("no base URL found for itag: %s", itag)
		}
		baseURL, err := url.Parse(baseURLString)
		if err != nil {
			return fmt.Errorf("parsing base URL: %w", err)
		}

		// Extract the sq parameter
		sq := urlutil.ExtractParameter(req.URL.Path, "sq")
		if sq == "" {
			// Looks like a request for the head sequence number
			req.URL = baseURL
		} else {
			// Looks like as a request for a specific segment
			req.URL = urlutil.BuildSegmentURLFromParsed(baseURL, sq)
		}

		req.Header.Set("X-Request-Url", req.URL.String())

		return nil
	}

	client.Logger = nil

	return client
}

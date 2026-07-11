package urlutil

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildVideoURL(id string) string {
	return "https://www.youtube.com/watch?v=" + id
}

func BuildVideoShortURL(id string) string {
	return "https://www.youtu.be/watch?v=" + id
}

func BuildVideoLiveURL(id string) string {
	return "https://www.youtube.com/live/" + id
}

func BuildSegmentURL(baseURL, sq string) (*url.URL, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	return BuildSegmentURLFromParsed(base, sq), nil
}

func BuildSegmentURLFromParsed(baseURL *url.URL, sq string) *url.URL {
	result := *baseURL
	query := result.Query()
	query.Set("sq", sq)
	result.RawQuery = query.Encode()
	return &result
}

func ExtractParameter(rawURL, name string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return u.Query().Get(name)
}

func FormatServerAddress(addr string) string {
	parts := strings.Split(addr, ":")
	host, port := parts[0], parts[1]
	if host == "" {
		return "http://localhost:" + port
	}
	return "http://" + addr
}

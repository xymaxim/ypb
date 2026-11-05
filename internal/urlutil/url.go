package urlutil

import (
	"fmt"
	"net"
	"net/url"
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

// FormatServerAddress returns the base URL of a server listening on addr
// ("host:port"): printed for the user, and used in the URLs that yt-dlp and
// players connect to (the manifest URL for download, BaseURL in manifests).
// All-interfaces addresses are given as localhost, and IPv6 hosts are
// bracketed. The server itself listens on addr as given.
func FormatServerAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	ip := net.ParseIP(host)
	if host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

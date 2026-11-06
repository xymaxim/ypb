package urlutil

import (
	"fmt"
	"net"
	"net/netip"
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

// FormatServerAddress returns the URL to print for a server listening on addr
// ("host:port"). All-interfaces addresses are shown as localhost, and IPv6
// hosts are bracketed. Display only: the server listens on addr as given.
func FormatServerAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	ip, parseErr := netip.ParseAddr(host)
	if host == "" || (parseErr == nil && ip.Unmap().IsUnspecified()) {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

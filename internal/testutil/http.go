package testutil

import (
	"net/http"
	"net/url"
)

type rewriteTransport struct {
	host     string
	underlay http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.URL.Scheme = "http"
	r.URL.Host = rt.host
	return rt.underlay.RoundTrip(r)
}

func NewClient(addr string) *http.Client {
	u, _ := url.Parse(addr)
	return &http.Client{
		Transport: &rewriteTransport{
			host:     u.Host,
			underlay: http.DefaultTransport,
		},
	}
}

func MakeDummyHandler() http.Handler {
	return http.HandlerFunc(
		func(_ http.ResponseWriter, _ *http.Request) {},
	)
}

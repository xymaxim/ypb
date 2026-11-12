package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// WithCORS allows the given origins to read responses. Origins are matched
// exactly and no wildcard origin is sent. An empty set disables CORS.
func WithCORS(allowed map[string]bool, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Add("Vary", "Origin")

		origin := r.Header.Get("Origin")
		isPreflight := r.Method == http.MethodOptions &&
			origin != "" &&
			r.Header.Get("Access-Control-Request-Method") != ""

		if isPreflight {
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
		}

		if origin != "" && allowed[origin] {
			h.Set("Access-Control-Allow-Origin", origin)

			if isPreflight {
				h.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Range")
				h.Set("Access-Control-Max-Age", "600")
			} else {
				h.Set(
					"Access-Control-Expose-Headers",
					"Content-Range, Accept-Ranges",
				)
			}
		}

		if isPreflight {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewOriginSet builds the allowlist for WithCORS from scheme://host[:port]
// origins. Case and a trailing slash are normalized, and blank entries are
// skipped.
func NewOriginSet(origins []string) (map[string]bool, error) {
	allowed := make(map[string]bool, len(origins))

	for _, raw := range origins {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}

		u, err := url.Parse(o)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("invalid CORS origin %q", o)
		}

		origin := strings.ToLower(u.Scheme + "://" + u.Host)
		if strings.TrimSuffix(strings.ToLower(o), "/") != origin {
			return nil, fmt.Errorf("invalid CORS origin %q", o)
		}

		allowed[origin] = true
	}

	return allowed, nil
}

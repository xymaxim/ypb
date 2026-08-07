package app

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed web/index.html
var webFS embed.FS

type PlayHandler struct{}

func (h *PlayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	page, err := webFS.ReadFile("web/index.html")
	if err != nil {
		return fmt.Errorf("reading player page: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(page); err != nil {
		return fmt.Errorf("writing player page: %w", err)
	}
	return nil
}

package app

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

type PlayHandler struct{}

func (h *PlayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	content, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("creating web filesystem: %w", err)
	}
	http.FileServer(http.FS(content)).ServeHTTP(w, r)
	return nil
}

func (h *PlayHandler) ServePage(w http.ResponseWriter, r *http.Request) error {
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		return fmt.Errorf("reading player page: %w", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write(data)
	return err
}

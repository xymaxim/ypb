package app

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed ui/dist
var uiFS embed.FS

type PlayHandler struct{}

func (h *PlayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	content, err := fs.Sub(uiFS, "ui/dist")
	if err != nil {
		return fmt.Errorf("creating ui filesystem: %w", err)
	}
	http.FileServer(http.FS(content)).ServeHTTP(w, r)
	return nil
}

func (h *PlayHandler) ServePage(w http.ResponseWriter, r *http.Request) error {
	data, err := uiFS.ReadFile("ui/dist/index.html")
	if err != nil {
		return fmt.Errorf("reading player page: %w", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write(data)
	return err
}

func (h *PlayHandler) ServeRoot(w http.ResponseWriter, r *http.Request) error {
	http.Redirect(w, r, "/now", http.StatusMovedPermanently)
	return nil
}

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

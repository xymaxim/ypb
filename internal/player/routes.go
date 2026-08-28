package player

import (
	"net/http"

	apppkg "github.com/xymaxim/ypb/internal/app"
)

func RegisterPlayRoutes(mux *http.ServeMux) {
	mux.Handle("/{$}", apppkg.WithError((&PlayHandler{}).ServeRoot))
	mux.Handle("/", apppkg.WithError((&PlayHandler{}).ServeHTTP))
	mux.Handle("/{interval}", apppkg.WithError((&PlayHandler{}).ServePage))
}

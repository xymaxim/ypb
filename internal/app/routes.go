package app

import "net/http"

const (
	InfoPath    = "/info"
	MPDPath     = "/mpd/{interval}"
	SegmentPath = "/segments/itag/{itag}/sq/{sq}"
)

func RegisterInfoRoute(mux *http.ServeMux, app *App) {
	mux.HandleFunc(InfoPath, WithError((&InfoHandler{Info: app.Playback.Info()}).ServeHTTP))
}

func RegisterSegmentRoute(mux *http.ServeMux, app *App) {
	mux.HandleFunc(SegmentPath, WithError(
		(&SegmentHandler{Playback: app.Playback}).ServeHTTP),
	)
}

func RegisterMPDRoute(mux *http.ServeMux, app *App) {
	mux.HandleFunc(MPDPath, WithError((&MPDHandler{
		Playback:      app.Playback,
		FFprobeRunner: app.FFprobeRunner,
		ServerAddr:    app.Server.Addr,
	}).ServeHTTP))
}

package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestPlayRootRedirectsToNow(t *testing.T) {
	t.Parallel()

	h := &app.PlayHandler{}

	mux := http.NewServeMux()
	mux.Handle("/{$}", app.WithError(h.ServeRoot))
	mux.Handle("/", app.WithError(h.ServeHTTP))
	mux.Handle("/{interval}", app.WithError(h.ServePage))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/now", w.Header().Get("Location"))
}

func TestPlayIntervalPathServesPage(t *testing.T) {
	t.Parallel()

	h := &app.PlayHandler{}

	mux := http.NewServeMux()
	mux.Handle("/", app.WithError(h.ServeHTTP))
	mux.Handle("/{interval}", app.WithError(h.ServePage))

	for _, path := range []string{"/03:00+30s", "/30m--now", "/anything"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "path %s", path)
		require.Contains(t, w.Header().Get("Content-Type"), "text/html", "path %s", path)
	}
}

func TestPlayRootDoesNotShadowAPI(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/{$}", app.WithError((&app.PlayHandler{}).ServeRoot))
	mux.Handle("/", app.WithError((&app.PlayHandler{}).ServeHTTP))
	mux.Handle("/{interval}", app.WithError((&app.PlayHandler{}).ServePage))
	mux.HandleFunc(app.InfoPath, app.WithError(
		(&app.InfoHandler{Info: testutil.SampleVideoInfo()}).ServeHTTP),
	)

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "application/json")

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/now", w.Header().Get("Location"))

	req = httptest.NewRequest(http.MethodGet, "/03:00", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

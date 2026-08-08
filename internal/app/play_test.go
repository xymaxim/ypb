package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/app"
	"github.com/xymaxim/ypb/internal/testutil"
)

func TestPlayHandlerUnknownPathNotFound(t *testing.T) {
	t.Parallel()

	h := &app.PlayHandler{}

	mux := http.NewServeMux()
	mux.Handle("/", app.WithError(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestPlayRootDoesNotShadowAPI(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/", app.WithError((&app.PlayHandler{}).ServeHTTP))
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

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

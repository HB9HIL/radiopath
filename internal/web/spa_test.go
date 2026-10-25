package web

import (
	"log/slog"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPA(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":            {Data: []byte("spa")},
		"_app/immutable/app.js": {Data: []byte("js")},
		"_app/version.json":     {Data: []byte("{}")},
	}
	h := (&Server{Log: slog.Default(), UI: ui}).Handler()
	for _, tc := range []struct {
		method, path string
		code         int
		body, cache  string
	}{
		{"GET", "/does/not/exist", 200, "spa", "no-cache"},
		{"GET", "/_app/immutable/app.js", 200, "js", "public, max-age=31536000, immutable"},
		{"GET", "/_app/version.json", 200, "{}", "no-cache"},
		{"GET", "/_app/immutable/missing.js", 404, "", ""},
		{"GET", "/missing.png", 404, "", ""},
		{"GET", "/api/missing", 404, "", ""},
		{"POST", "/does/not/exist", 404, "", ""},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code || (tc.body != "" && w.Body.String() != tc.body) || (tc.cache != "" && w.Header().Get("Cache-Control") != tc.cache) {
			t.Errorf("%s %s: %d %q cache %q", tc.method, tc.path, w.Code, w.Body.String(), w.Header().Get("Cache-Control"))
		}
	}
}

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteLabel(t *testing.T) {
	nop := func(http.ResponseWriter, *http.Request) {}
	app := http.NewServeMux()
	app.HandleFunc("GET /sites/{id}", nop)
	api := http.NewServeMux()
	api.HandleFunc("GET /api/password/reset/{token}", nop)
	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", nop)
	root.Handle("/api/", api)
	root.HandleFunc("GET /tiles/{z}/{x}/{y}", nop)
	root.Handle("/s/", http.HandlerFunc(nop))
	root.Handle("/", app)

	for _, tc := range []struct{ path, want string }{
		{"/healthz", "/healthz"},
		{"/api/password/reset/s3cret-token", "/api/password/reset/{token}"},
		{"/api/nothing", "other"},
		{"/sites/42", "/sites/{id}"},
		{"/tiles/12/2145/1436.png", "/tiles/{z}/{x}/{y}"},
		{"/s/l/s3cret-token", "/s/"},
		{"/nothing/here", "other"},
	} {
		got := routeLabel(root, map[string]*http.ServeMux{"/": app, "/api/": api}, httptest.NewRequest("GET", tc.path, nil))
		if got != tc.want {
			t.Errorf("routeLabel(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

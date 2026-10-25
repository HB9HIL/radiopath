package web

import (
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http/httptest"
	"strings"
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

// The SPA and the admin pages allow exactly their own inline scripts, by hash, and no other inline script.
func TestCSP(t *testing.T) {
	script := "document.documentElement.dataset.theme = 'dark'"
	s := &Server{Log: slog.Default(), UI: fstest.MapFS{"index.html": {Data: []byte("<head><script>" + script + "</script></head>")}}}
	h := s.Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/sites", nil))
	sum := sha256.Sum256([]byte(script))
	policy := w.Header().Get("Content-Security-Policy")
	scripts := strings.SplitN(strings.SplitN(policy, "script-src ", 2)[1], ";", 2)[0]
	if scripts != "'self' 'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'" {
		t.Errorf("SPA script-src %q", scripts)
	}
	for _, want := range []string{"default-src 'self'", "object-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(policy, want) {
			t.Errorf("SPA policy lacks %s: %s", want, policy)
		}
	}
	if strings.Contains(s.adminCSP, "script-src 'self'") {
		t.Errorf("admin policy allows script files: %s", s.adminCSP)
	}
	// every script the admin pages really send must be allowed
	w = httptest.NewRecorder()
	s.render(w, 200, "admin_users", adminData{base: base{Admin: true}, PendingUser: "op"})
	for _, m := range inlineScript.FindAllStringSubmatch(w.Body.String(), -1) {
		sum := sha256.Sum256([]byte(m[1]))
		if !strings.Contains(s.adminCSP, base64.StdEncoding.EncodeToString(sum[:])) {
			t.Errorf("admin script not in the policy: %.60s", m[1])
		}
	}
	if w.Header().Get("Content-Security-Policy") != s.adminCSP {
		t.Error("admin page without its policy")
	}
}

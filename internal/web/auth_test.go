package web

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidateCredentialsLength(t *testing.T) {
	if errs := ValidateCredentials("hb9hil", strings.Repeat("x", 72)); len(errs) != 0 {
		t.Errorf("72 bytes rejected: %v", errs)
	}
	if errs := ValidateCredentials("hb9hil", strings.Repeat("x", 73)); len(errs) != 1 {
		t.Errorf("73 bytes: want one error, got %v", errs)
	}
}

func TestFormTextLength(t *testing.T) {
	long := strings.Repeat("ä", maxTextLen+1)
	for _, tc := range []struct {
		value string
		errs  int
	}{{"HB9HIL", 0}, {strings.Repeat("ä", maxTextLen), 0}, {long, 1}, {"", 1}} {
		r := httptest.NewRequest("POST", "/sites", strings.NewReader(url.Values{"name": {tc.value}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		f := &form{r: r}
		f.text("name", "Name")
		if len(f.Errors) != tc.errs {
			t.Errorf("text(%d runes): errors %v, want %d", len([]rune(tc.value)), f.Errors, tc.errs)
		}
	}
}

func TestSecureHeaders(t *testing.T) {
	h := (&Server{Log: slog.Default()}).Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if w.Code != http.StatusOK {
		t.Errorf("healthz: %d", w.Code)
	}
}

func TestValidEmail(t *testing.T) {
	for in, want := range map[string]string{
		"op@hb9hil.example":                      "op@hb9hil.example",
		"  op@hb9hil.example ":                   "op@hb9hil.example",
		"Op <op@hb9hil.example>":                 "",
		"op@hb9hil.example, x@y.z":               "",
		"not-an-address":                         "",
		"":                                       "",
		"op@" + strings.Repeat("a", 250) + ".ch": "",
	} {
		if got := validEmail(in); got != want {
			t.Errorf("validEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMailRoutesNeedMailer(t *testing.T) {
	h := (&Server{Log: slog.Default()}).Handler()
	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/api/verify", `{"token":"abc"}`},
		{"POST", "/api/verify/resend", `{"name":"hb9hil"}`},
		{"POST", "/api/password/forgot", `{"email":"op@example.org"}`},
		{"GET", "/api/password/reset/abc", ""},
		{"POST", "/api/password/reset", `{"token":"abc","password":"0123456789"}`},
		{"POST", "/api/register", `{"name":"hb9hil","email":"op@example.org","password":"0123456789"}`},
	} {
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s %s without mailer: %d %s", tc.method, tc.path, w.Code, w.Header().Get("Content-Type"))
		}
	}
}

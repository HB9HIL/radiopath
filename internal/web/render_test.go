package web

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/radiopath/radiopath/internal/store"
)

func TestRenderPages(t *testing.T) {
	s := &Server{Log: slog.Default(), Registration: true, Version: "0.1.0-42-gdeadbee"}
	s.Handler()

	now := time.Now()

	cases := []struct {
		page string
		data any
	}{
		{"notfound", base{}},
		{"admin_login", adminLoginData{base: base{Admin: true}, Error: "Wrong token."}},
		{"admin_users", adminData{base: base{Admin: true}, Notice: "hb9hil: 2 session(s) revoked",
			Users:       []store.UserStat{{User: store.User{ID: 1, Name: "hb9hil", CreatedAt: now}, Sessions: 2, LastLogin: &now}},
			PendingUser: "hb9hil", Maintenance: true}},
		{"admin_users", adminData{base: base{Admin: true}, Errors: []string{"No such user: nope"}}},
	}

	outDir := os.Getenv("RADIOPATH_TEST_OUT")

	for i, c := range cases {
		var b strings.Builder
		if err := s.tmpl[c.page].Execute(&b, c.data); err != nil {
			t.Fatalf("%s: %v", c.page, err)
		}
		out := b.String()
		if outDir != "" {
			name := fmt.Sprintf("%s/%02d_%s.html", outDir, i, c.page)
			if err := os.WriteFile(name, []byte(out), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if strings.Contains(out, "<no value>") {
			t.Errorf("%s: rendered a missing value", c.page)
		}
		for _, want := range []string{`href="/static/app.css`, "cp-theme", "</html>"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q", c.page, want)
			}
		}
	}
}

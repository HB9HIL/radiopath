package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/radiopath/radiopath/internal/store"
)

// Operations anyone may call; every other one must declare security, so a new operation is never public by accident.
var publicOps = map[string]bool{
	"get-config": true, "create-session": true, "delete-session": true, "register": true, "verify-email": true,
	"resend-verification": true, "forgot-password": true, "check-reset-token": true, "reset-password": true,
	"get-shared-link": true, "get-shared-coverage": true,
}

func TestAPISecurity(t *testing.T) {
	spec, err := (&Server{}).OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Security    []map[string][]string `json:"security"`
		}
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	for path, ops := range doc.Paths {
		for method, o := range ops {
			if public := publicOps[o.OperationID]; public == (len(o.Security) > 0) {
				t.Errorf("%s %s (%s): public %v but security %v", method, path, o.OperationID, public, o.Security)
			}
		}
	}

	h := (&Server{Log: slog.Default()}).Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/me", nil))
	if w.Code != http.StatusUnauthorized || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Errorf("GET /api/me without session: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
}

// The API views never carry the owner, give empty lists as [] and point shared rasters at the token.
func TestAPIViews(t *testing.T) {
	s := &Server{BaseURL: "https://radiopath.example"}
	now := time.Now()
	l := store.LinkWithSites{Link: store.Link{ID: 1, OwnerID: 7, Name: "a to b", ShareToken: "tok", ShareExpires: &now}}
	c := store.CoverageWithSite{Coverage: store.Coverage{ID: 2, OwnerID: 7, Name: "area", Job: store.Job{State: store.JobIdle},
		ComputedAt: &now, Bounds: &store.Bounds{North: 1}}}
	link := toLinkDetail(linkData{Link: l, Error: "no terrain"}, s.shareURL("/s/l/"))
	cov := s.coverageDetail(context.Background(), c, "/s/c/tok", []Overlay{})
	for name, v := range map[string]any{"link": link, "coverage": cov} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "owner") {
			t.Errorf("%s carries the owner: %s", name, b)
		}
	}
	b, _ := json.Marshal(cov)
	for _, want := range []string{`"overlays":[]`, `"image_url":"/s/c/tok/image.png?t=`, `"composite_url":"/s/c/tok/composite.png"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("coverage: missing %s in %s", want, b)
		}
	}
	if link.Share == nil || link.Share.URL != "https://radiopath.example/s/l/tok" {
		t.Errorf("share: %+v", link.Share)
	}
}

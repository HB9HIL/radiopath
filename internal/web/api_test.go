package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
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

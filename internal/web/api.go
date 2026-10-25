package web

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/radiopath/radiopath/internal/store"
)

// init makes empty lists [] instead of null in every response and schema.
func init() {
	huma.DefaultArrayNullable = false
}

// Scopes name what an operation needs; for now any session grants all of them. scopeSession marks operations no API token may call.
const (
	scopeAccountRead = "account:read"
	scopeSession     = "session"
)

// api builds the JSON API on its own mux; Handler mounts it under /api/ behind authenticate.
func (s *Server) api() *http.ServeMux {
	mux := http.NewServeMux()
	s.registerAPI(humago.New(mux, s.apiConfig()))
	return mux
}

// OpenAPI returns the API description as JSON without needing a database.
func (s *Server) OpenAPI() ([]byte, error) {
	api := humago.New(http.NewServeMux(), s.apiConfig())
	s.registerAPI(api)
	return api.OpenAPI().MarshalJSON()
}

// apiConfig returns the huma configuration with the API paths and the session scheme.
func (s *Server) apiConfig() huma.Config {
	c := huma.DefaultConfig("Radiopath API", s.Version)
	c.OpenAPIPath = "/api/openapi"
	c.DocsPath = "/api/docs"
	c.SchemasPath = "/api/schemas"
	c.CreateHooks = nil
	c.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: sessionCookie, Description: "The session cookie set by POST /api/session."},
	}
	return c
}

// registerAPI registers the middleware and all API operations.
func (s *Server) registerAPI(api huma.API) {
	api.UseMiddleware(s.gate(api))
	s.registerConfig(api)
	s.registerAuth(api)
	s.registerMailFlows(api)
	s.registerSites(api)
	s.registerAntennas(api)
	s.registerLinks(api)
	s.registerCoverages(api)
	s.registerAccount(api)
}

// op returns an operation; a non-empty scope requires a logged-in user, an empty one makes it public.
func op(id, method, path, scope, summary string) huma.Operation {
	o := huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary}
	if scope != "" {
		o.Security = []map[string][]string{{"session": {}}}
		o.Errors = []int{http.StatusUnauthorized}
	}
	return o
}

// gate answers 401 for operations with security when nobody is logged in.
func (s *Server) gate(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if len(ctx.Operation().Security) > 0 && userOf(ctx.Context()).ID == 0 {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Not logged in")
			return
		}
		next(ctx)
	}
}

type TokenPath struct {
	Token string `path:"token"`
}

type requestInfo struct {
	ip     string
	secure bool
	cookie string
}

const reqKey ctxKey = 1

// authenticate puts the request info and the session user, if any, into the context; it never rejects, gate does.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := requestInfo{ip: s.clientIP(r), secure: secureCookie(r)}
		if c, err := r.Cookie(sessionCookie); err == nil {
			info.cookie = c.Value
		}
		ctx := context.WithValue(r.Context(), reqKey, info)
		u, err := s.sessionUser(r)
		switch {
		case err == nil:
			ctx = context.WithValue(ctx, userKey, u)
		case !errors.Is(err, store.ErrNotFound):
			s.Log.Error("session lookup", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// reqInfo returns what authenticate stored about the request.
func reqInfo(ctx context.Context) requestInfo {
	i, _ := ctx.Value(reqKey).(requestInfo)
	return i
}

// userOf returns the logged-in user of ctx, or the zero value.
func userOf(ctx context.Context) store.User {
	u, _ := ctx.Value(userKey).(store.User)
	return u
}

// check collects the invalid fields of an API input; err returns them as one 422.
type check struct{ errs []error }

// add records a problem with field; an empty field means the input as a whole.
func (c *check) add(field, msg string) {
	loc := "body"
	if field != "" {
		loc += "." + field
	}
	c.errs = append(c.errs, &huma.ErrorDetail{Location: loc, Message: msg})
}

// num checks that v lies within lo..hi and returns it.
func (c *check) num(field, label string, v, lo, hi float64) float64 {
	if math.IsNaN(v) || v < lo || v > hi {
		c.add(field, fmt.Sprintf("%s: must be between %g and %g", label, lo, hi))
	}
	return v
}

// optNum is num for an optional value; nil passes.
func (c *check) optNum(field, label string, v *float64, lo, hi float64) *float64 {
	if v != nil {
		c.num(field, label, *v, lo, hi)
	}
	return v
}

// deg checks an optional bearing in 0..360 and wraps 360 to 0; nil passes.
func (c *check) deg(field, label string, v *float64) *float64 {
	if v == nil {
		return nil
	}
	d := math.Mod(c.num(field, label, *v, 0, 360), 360)
	return &d
}

// text checks a required value of at most maxTextLen characters and returns it.
func (c *check) text(field, label, v string) string {
	if v == "" {
		c.add(field, label+": required")
	} else if utf8.RuneCountInString(v) > maxTextLen {
		c.add(field, fmt.Sprintf("%s: at most %d characters", label, maxTextLen))
	}
	return v
}

// err returns the collected problems as a 422, or nil.
func (c *check) err() error {
	if len(c.errs) == 0 {
		return nil
	}
	return huma.Error422UnprocessableEntity("Invalid input", c.errs...)
}

// storeErr maps store errors to problem responses and logs anything unexpected as a 500.
func (s *Server) storeErr(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return huma.Error404NotFound("Not found")
	default:
		s.Log.Error("request failed", "err", err)
		return huma.Error500InternalServerError("Internal error")
	}
}

type Config struct {
	Version       string `json:"version"`
	Registration  bool   `json:"registration" doc:"Whether new accounts can register"`
	PasswordReset bool   `json:"password_reset" doc:"Whether forgotten passwords can be reset by mail"`
	Maintenance   bool   `json:"maintenance" doc:"Logins are disabled while true"`
}

type ConfigOutput struct{ Body Config }

// registerConfig registers GET /api/config, what the login pages need to know before anyone is logged in.
func (s *Server) registerConfig(api huma.API) {
	huma.Register(api, op("get-config", http.MethodGet, "/api/config", "", "Server version and enabled features"),
		func(ctx context.Context, _ *struct{}) (*ConfigOutput, error) {
			on, err := s.Store.Maintenance(ctx)
			if err != nil {
				return nil, s.storeErr(err)
			}
			return &ConfigOutput{Body: Config{Version: s.Version, Registration: s.Registration, PasswordReset: s.Mail != nil, Maintenance: on}}, nil
		})
}

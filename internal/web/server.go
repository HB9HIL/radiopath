package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/radiopath/radiopath/internal/link"
	"github.com/radiopath/radiopath/internal/mail"
	"github.com/radiopath/radiopath/internal/store"
	"github.com/radiopath/radiopath/internal/tiles"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var pages = []string{"sites_list", "site_form", "links_list", "link_form", "link_detail",
	"coverages_list", "coverage_form", "coverage_detail", "antennas_list", "login",
	"register", "account", "forgot", "reset", "admin_login", "admin_users", "notfound"}

type base struct {
	User  string
	Admin bool
}

// base returns the template data shared by all logged-in pages.
func (s *Server) base(r *http.Request) base { return base{User: currentUser(r).Name} }

type Server struct {
	Store        *store.Store
	Analyzer     link.Analyzer
	Tiles        *tiles.Proxy
	DEMReady     func() error
	Registration bool
	Mail         mail.Mailer
	BaseURL      string
	TrustProxy   bool
	AdminToken   string
	Version      string
	Log          *slog.Logger
	// UI holds the built SPA (webui.FS()), served for every path no route claims.
	UI fs.FS

	tmpl map[string]*template.Template
}

// Handler parses the templates and builds the full route tree: public, share, optional admin, registration and mail routes, and everything else behind requireAuth.
func (s *Server) Handler() http.Handler {
	funcs := template.FuncMap{
		"divKm":        func(m float64) float64 { return m / 1000 },
		"offBoresight": link.OffBoresight,
		"asset":        assetURL,
		"isID":         func(id int64, ref *int64) bool { return ref != nil && *ref == id },
		"version":      func() string { return s.Version },
	}
	s.tmpl = make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		s.tmpl[p] = template.Must(template.New("layout.html").Funcs(funcs).
			ParseFS(templateFS, "templates/layout.html", "templates/icons.html", "templates/partials.html",
				"templates/"+p+".html"))
	}

	if s.Tiles != nil {
		s.Tiles.ClientIP = s.clientIP
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		n, err := s.Store.CountSites(r.Context(), currentUser(r).ID)
		if err != nil {
			s.fail(w, err)
			return
		}
		if n == 0 {
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/links", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /logout", s.logout)

	mux.HandleFunc("GET /sites", s.listSites)
	mux.HandleFunc("GET /sites/new", s.newSite)
	mux.HandleFunc("POST /sites", s.createSite)
	mux.HandleFunc("GET /sites/{id}", s.editSite)
	mux.HandleFunc("POST /sites/{id}", s.updateSite)
	mux.HandleFunc("POST /sites/{id}/delete", s.deleteSite)

	mux.HandleFunc("GET /antennas", s.listAntennas)
	mux.HandleFunc("POST /antennas", s.createAntenna)
	mux.HandleFunc("POST /antennas/manual", s.createManualAntenna)
	mux.HandleFunc("POST /antennas/{id}", s.updateAntenna)
	mux.HandleFunc("POST /antennas/{id}/delete", s.deleteAntenna)

	mux.HandleFunc("GET /links", s.listLinks)
	mux.HandleFunc("GET /links/new", s.newLink)
	mux.HandleFunc("POST /links", s.createLink)
	mux.HandleFunc("GET /links/{id}", s.showLink)
	mux.HandleFunc("GET /links/{id}/edit", s.editLink)
	mux.HandleFunc("GET /links/{id}/report.pdf", s.linkReport)
	mux.HandleFunc("POST /links/{id}", s.updateLink)
	mux.HandleFunc("POST /links/{id}/delete", s.deleteLink)
	mux.HandleFunc("POST /links/{id}/share", s.shareLink)
	mux.HandleFunc("POST /links/{id}/share/delete", s.unshareLink)

	mux.HandleFunc("GET /account", s.account)
	mux.HandleFunc("POST /account/email", s.accountEmail)
	mux.HandleFunc("POST /account/password", s.accountPassword)

	mux.HandleFunc("GET /coverages", s.listCoverages)
	mux.HandleFunc("GET /coverages/new", s.newCoverage)
	mux.HandleFunc("POST /coverages", s.createCoverage)
	mux.HandleFunc("GET /coverages/{id}", s.showCoverage)
	mux.HandleFunc("GET /coverages/{id}/edit", s.editCoverage)
	mux.HandleFunc("GET /coverages/{id}/report.pdf", s.coverageReport)
	mux.HandleFunc("POST /coverages/{id}", s.updateCoverage)
	mux.HandleFunc("POST /coverages/{id}/delete", s.deleteCoverage)
	mux.HandleFunc("POST /coverages/{id}/compute", s.computeCoverage)
	mux.HandleFunc("GET /coverages/{id}/image.png", s.coverageImage)
	mux.HandleFunc("GET /coverages/{id}/composite.png", s.coverageComposite)
	mux.HandleFunc("GET /coverages/{id}/events", s.coverageEvents)
	mux.HandleFunc("POST /coverages/{id}/view", s.setCoverageView)
	mux.HandleFunc("POST /coverages/{id}/legend", s.setCoverageLegend)
	mux.HandleFunc("POST /coverages/{id}/share", s.shareCoverage)
	mux.HandleFunc("POST /coverages/{id}/share/delete", s.unshareCoverage)

	root := http.NewServeMux()
	if s.adminEnabled() {
		admin := http.NewServeMux()
		admin.HandleFunc("GET /admin/{$}", s.adminIndex)
		admin.HandleFunc("POST /admin/login", s.adminLogin)
		admin.HandleFunc("POST /admin/logout", s.adminLogout)
		admin.Handle("GET /admin/users/{name}/password", s.requireAdmin(http.HandlerFunc(s.adminPasswordForm)))
		admin.Handle("POST /admin/maintenance", s.requireAdmin(http.HandlerFunc(s.adminMaintenance)))
		admin.Handle("POST /admin/sessions/delete", s.requireAdmin(http.HandlerFunc(s.adminRevokeAllSessions)))
		admin.Handle("POST /admin/users", s.requireAdmin(http.HandlerFunc(s.adminCreateUser)))
		admin.Handle("POST /admin/users/{name}/password", s.requireAdmin(http.HandlerFunc(s.adminSetPassword)))
		admin.Handle("POST /admin/users/{name}/sessions", s.requireAdmin(http.HandlerFunc(s.adminRevokeSessions)))
		admin.Handle("POST /admin/users/{name}/delete", s.requireAdmin(http.HandlerFunc(s.adminDeleteUser)))
		admin.HandleFunc("/", s.notFound)
		root.HandleFunc("GET /admin", s.adminIndex)
		root.Handle("/admin/", admin)
	}
	share := http.NewServeMux()
	share.HandleFunc("GET /s/l/{token}", s.sharedLink)
	share.HandleFunc("GET /s/l/{token}/report.pdf", s.sharedLinkReport)
	share.HandleFunc("GET /s/c/{token}", s.sharedCoverage)
	share.HandleFunc("GET /s/c/{token}/report.pdf", s.sharedCoverageReport)
	share.HandleFunc("GET /s/c/{token}/image.png", s.sharedCoverageImage)
	share.HandleFunc("GET /s/c/{token}/composite.png", s.sharedCoverageComposite)
	share.HandleFunc("/", s.notFound)
	root.Handle("/s/", s.shareLimit(share))
	root.Handle("GET /tiles/{z}/{x}/{y}", s.Tiles)
	root.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("User-agent: *\nDisallow: /tiles/\n"))
	})
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok\n")) })
	root.HandleFunc("GET /readyz", s.readyz)
	root.Handle("GET /static/", staticHandler())
	root.HandleFunc("GET /login", s.loginForm)
	root.HandleFunc("POST /login", s.login)
	if s.Registration {
		root.HandleFunc("GET /register", s.registerForm)
		root.HandleFunc("POST /register", s.register)
	}
	if s.Mail != nil {
		root.HandleFunc("GET /forgot", s.forgotForm)
		root.HandleFunc("POST /forgot", s.forgot)
		root.HandleFunc("GET /reset/{token}", s.resetForm)
		root.HandleFunc("POST /reset/{token}", s.reset)
		root.HandleFunc("GET /verify/{token}", s.verify)
		root.HandleFunc("POST /verify/resend", s.resendVerification)
	}
	// ponytail: template routes go first until each section is ported to the SPA, then "/" serves only the SPA.
	app, spa := s.requireAuth(mux), s.spa()
	root.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, p := mux.Handler(r); p != "" {
			app.ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	}))

	return secureHeaders(instrument(root, mux, http.NewCrossOriginProtection().Handler(root)))
}

// secureHeaders adds the nosniff, frame and referrer headers to every response.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// spa serves the SvelteKit build: existing files as they are, any other path without a file extension the index.html fallback.
// Unknown API paths, _app assets and files answer 404, so a stale bundle fails loudly instead of loading HTML.
func (s *Server) spa() http.Handler {
	files := http.FileServerFS(s.UI)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(s.UI, name); err != nil {
			if strings.HasPrefix(name, "_app/") || strings.Contains(path.Base(name), ".") {
				http.NotFound(w, r)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		if strings.HasPrefix(name, "_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

// readyz answers GET /readyz with 200 only if the database, the DEM data and the tile cache are usable, otherwise 503 naming the failing part.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2e9)
	defer cancel()
	if err := s.Store.Ping(ctx); err != nil {
		http.Error(w, "database: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err := s.DEMReady(); err != nil {
		http.Error(w, "dem: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err := s.Tiles.Ready(ctx); err != nil {
		http.Error(w, "tile cache: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok\n"))
}

var staticETags = func() map[string]string {
	etags := map[string]string{}
	fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}
		etags["/"+path] = tiles.ETag(b)
		return nil
	})
	return etags
}()

// assetURL appends the content hash to a static path so browsers can cache it forever; unknown paths come back unchanged.
func assetURL(path string) string {
	etag, ok := staticETags[path]
	if !ok {
		return path
	}
	return path + "?v=" + strings.Trim(etag, `"`)
}

// staticHandler serves the embedded static files with ETags, immutable caching for hashed URLs and 304 on a match.
func staticHandler() http.Handler {
	files := http.StripPrefix("/", http.FileServerFS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etag, ok := staticETags[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("v") == strings.Trim(etag, `"`) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		files.ServeHTTP(w, r)
	})
}

// render executes the named page template with the given status.
func (s *Server) render(w http.ResponseWriter, status int, page string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tmpl[page].Execute(w, data); err != nil {
		s.Log.Error("render", "page", page, "err", err)
	}
}

// notFound renders the 404 page.
func (s *Server) notFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, http.StatusNotFound, "notfound", base{})
}

// fail maps store.ErrNotFound to the 404 page and logs anything else, answering 500 without details.
func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.notFound(w, nil)
	default:
		s.Log.Error("request failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// escapeQuery URL-encodes v for use in a query string.
func escapeQuery(v string) string { return url.QueryEscape(v) }

// pathID parses the {id} path value.
func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// fromID parses the ?from= query value; ok is false unless it is a positive number.
func fromID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	return id, err == nil && id > 0
}

const maxTextLen = 100

type form struct {
	r      *http.Request
	Errors []string
}

// get returns the posted value for name.
func (f *form) get(name string) string { return f.r.PostFormValue(name) }

// num parses a required number within lo..hi; problems are collected in Errors and 0 or the out-of-range value is returned.
func (f *form) num(name, label string, lo, hi float64) float64 {
	v, err := strconv.ParseFloat(f.get(name), 64)
	if err != nil {
		f.Errors = append(f.Errors, label+": not a number")
		return 0
	}
	if v < lo || v > hi {
		f.Errors = append(f.Errors, fmt.Sprintf("%s: must be between %g and %g", label, lo, hi))
	}
	return v
}

// optNum is num for an optional field; empty gives 0.
func (f *form) optNum(name, label string, lo, hi float64) float64 {
	if f.get(name) == "" {
		return 0
	}
	return f.num(name, label, lo, hi)
}

// text returns a required value of at most maxTextLen characters, collecting an error otherwise.
func (f *form) text(name, label string) string {
	v := f.get(name)
	if v == "" {
		f.Errors = append(f.Errors, label+": required")
	} else if utf8.RuneCountInString(v) > maxTextLen {
		f.Errors = append(f.Errors, fmt.Sprintf("%s: at most %d characters", label, maxTextLen))
	}
	return v
}

// optID parses an optional ID; nil when empty or invalid (invalid also adds an error).
func (f *form) optID(name, label string) *int64 {
	v := f.get(name)
	if v == "" {
		return nil
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		f.Errors = append(f.Errors, label+": invalid")
		return nil
	}
	return &id
}

// optNumP is num for an optional field; nil when empty.
func (f *form) optNumP(name, label string, lo, hi float64) *float64 {
	if f.get(name) == "" {
		return nil
	}
	v := f.num(name, label, lo, hi)
	return &v
}

// optDeg parses an optional bearing in 0..360 and wraps 360 to 0; nil when empty.
func (f *form) optDeg(name, label string) *float64 {
	if f.get(name) == "" {
		return nil
	}
	v := math.Mod(f.num(name, label, 0, 360), 360)
	return &v
}

// id parses a required ID, collecting an error when invalid.
func (f *form) id(name, label string) int64 {
	v, err := strconv.ParseInt(f.get(name), 10, 64)
	if err != nil {
		f.Errors = append(f.Errors, label+": invalid")
	}
	return v
}

// values returns the posted fields as a flat map, e.g. to refill the form after errors.
func (f *form) values() map[string]string {
	out := make(map[string]string, len(f.r.PostForm))
	for k := range f.r.PostForm {
		out[k] = f.r.PostFormValue(k)
	}
	return out
}

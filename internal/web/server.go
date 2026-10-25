package web

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/radiopath/radiopath/internal/link"
	"github.com/radiopath/radiopath/internal/mail"
	"github.com/radiopath/radiopath/internal/store"
	"github.com/radiopath/radiopath/internal/tiles"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var pages = []string{"account", "admin_login", "admin_users", "notfound"}

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
		"asset":   assetURL,
		"version": func() string { return s.Version },
	}
	s.tmpl = make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		s.tmpl[p] = template.Must(template.New("layout.html").Funcs(funcs).
			ParseFS(templateFS, "templates/layout.html", "templates/icons.html",
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

	mux.HandleFunc("GET /links/{id}/report.pdf", s.linkReport)

	mux.HandleFunc("GET /account", s.account)
	mux.HandleFunc("POST /account/email", s.accountEmail)
	mux.HandleFunc("POST /account/password", s.accountPassword)

	mux.HandleFunc("GET /coverages/{id}/report.pdf", s.coverageReport)
	mux.HandleFunc("GET /coverages/{id}/image.png", s.coverageImage)
	mux.HandleFunc("GET /coverages/{id}/composite.png", s.coverageComposite)
	mux.HandleFunc("GET /coverages/{id}/events", s.coverageEvents)

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
	spa := s.spa()
	// Shared pages are SPA routes; the API request behind each one counts against the token's limit.
	sharedPage := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex")
		spa.ServeHTTP(w, r)
	})
	root.Handle("GET /s/l/{token}", sharedPage)
	root.Handle("GET /s/c/{token}", sharedPage)
	share := http.NewServeMux()
	share.HandleFunc("GET /s/l/{token}/report.pdf", s.sharedLinkReport)
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
	api := s.api()
	root.Handle("/api/", s.authenticate(api))
	// ponytail: template routes go first until each section is ported to the SPA, then "/" serves only the SPA.
	app := s.requireAuth(mux)
	root.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, p := mux.Handler(r); p != "" {
			app.ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	}))

	return secureHeaders(instrument(root, map[string]*http.ServeMux{"/": mux, "/api/": api}, http.NewCrossOriginProtection().Handler(root)))
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

// pathID parses the {id} path value.
func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

const maxTextLen = 100

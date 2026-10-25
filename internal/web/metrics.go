package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/radiopath/radiopath/internal/metrics"
)

// instrument records request count by route, method and status plus the duration for every request passing through next.
func instrument(root *http.ServeMux, subs map[string]*http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := routeLabel(root, subs, r)
		rec := &recorder{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)

		metrics.HTTPRequests.WithLabelValues(route, r.Method, strconv.Itoa(rec.code)).Inc()
		metrics.HTTPDuration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
}

// routeLabel returns the matched route pattern without its method as the metric label, so IDs and tokens do not blow up cardinality; unmatched requests give "other".
func routeLabel(root *http.ServeMux, subs map[string]*http.ServeMux, r *http.Request) string {
	_, pattern := root.Handler(r)
	if sub, ok := subs[pattern]; ok {
		_, pattern = sub.Handler(r)
	}
	if _, path, ok := strings.Cut(pattern, " "); ok {
		return path
	}
	if pattern == "" {
		return "other"
	}
	return pattern
}

type recorder struct {
	http.ResponseWriter
	code int
}

// WriteHeader remembers the status code for the metrics before passing it on.
func (r *recorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap exposes the wrapped writer so http.ResponseController can reach Flush and friends.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

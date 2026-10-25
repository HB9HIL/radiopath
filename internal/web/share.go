package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const shareTTL = 30 * 24 * time.Hour

const (
	shareWindow = time.Hour
	shareMax    = 120
)

// shareURL returns the absolute public URL for a share path.
func (s *Server) shareURL(path string) string {
	return s.BaseURL + path
}

// shareHit counts one request against the hourly limit of a share token; 429 over the limit.
func (s *Server) shareHit(ctx context.Context, token string) error {
	n, err := s.Store.RateHit(ctx, "share:"+token, shareWindow)
	if err != nil {
		return s.storeErr(err)
	}
	if n > shareMax {
		return huma.Error429TooManyRequests("Too many requests for this shared page, try again later")
	}
	return nil
}

// shareLimit rate limits public share URLs per token (404 for a missing token, 429 over the limit) and marks responses noindex and no-store.
func (s *Server) shareLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			s.notFound(w, r)
			return
		}
		if err := s.shareHit(r.Context(), parts[3]); err != nil {
			var se huma.StatusError
			if errors.As(err, &se) && se.GetStatus() == http.StatusTooManyRequests {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// shareCoverage handles POST /coverages/{id}/share: creates a share token valid for 30 days and redirects back to the coverage.
func (s *Server) shareCoverage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.notFound(w, r)
		return
	}
	if err := s.Store.ShareCoverage(r.Context(), currentUser(r).ID, id, randomToken(), time.Now().Add(shareTTL)); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/coverages/%d", id), http.StatusSeeOther)
}

// unshareCoverage handles POST /coverages/{id}/share/delete: removes the share token and redirects back to the coverage.
func (s *Server) unshareCoverage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.notFound(w, r)
		return
	}
	if err := s.Store.UnshareCoverage(r.Context(), currentUser(r).ID, id); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/coverages/%d", id), http.StatusSeeOther)
}

// sharedLinkReport serves the PDF report of a shared link.
func (s *Server) sharedLinkReport(w http.ResponseWriter, r *http.Request) {
	l, err := s.Store.GetSharedLink(r.Context(), r.PathValue("token"))
	if err != nil {
		s.fail(w, err)
		return
	}
	s.sendLinkReport(w, r, s.linkAnalysis(r.Context(), l), "")
}

// sharedCoverage renders the public read-only coverage page; it adds only the overlays the owner released for viewing, with token-scoped image URLs.
func (s *Server) sharedCoverage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	c, err := s.Store.GetSharedCoverage(r.Context(), token)
	if err != nil {
		s.fail(w, err)
		return
	}
	extra, err := s.Store.CoveragesByID(r.Context(), c.OwnerID, c.ViewOverlays)
	if err != nil {
		s.fail(w, err)
		return
	}
	data := s.coverageDetail(r, c, "/s/c/"+token)
	data.Shared = true
	for _, o := range extra {
		data.Others = append(data.Others, overlay{CoverageWithSite: o, Checked: true,
			URL: fmt.Sprintf("/s/c/%s/image.png?id=%d&t=%d", token, o.ID, o.ComputedAt.Unix())})
	}
	s.render(w, http.StatusOK, "coverage_detail", data)
}

// sharedCoverageReport serves the PDF report of a shared coverage.
func (s *Server) sharedCoverageReport(w http.ResponseWriter, r *http.Request) {
	c, err := s.Store.GetSharedCoverage(r.Context(), r.PathValue("token"))
	if err != nil {
		s.fail(w, err)
		return
	}
	s.sendCoverageReport(w, r, c, "")
}

// sharedCoverageComposite serves the overlay composite PNG of a shared coverage; 404 if a requested overlay is not released for viewing.
func (s *Server) sharedCoverageComposite(w http.ResponseWriter, r *http.Request) {
	c, err := s.Store.GetSharedCoverage(r.Context(), r.PathValue("token"))
	if err != nil {
		s.fail(w, err)
		return
	}
	with := withIDs(r.URL.Query().Get("with"))
	for _, n := range with {
		if !slices.Contains(c.ViewOverlays, n) {
			s.notFound(w, r)
			return
		}
	}
	s.composite(w, r, c.OwnerID, with, c.ID, legendOf(c.Coverage))
}

// sharedCoverageImage serves the PNG of a shared coverage or, via ?id=, of one of its released overlays; 404 for any other id.
func (s *Server) sharedCoverageImage(w http.ResponseWriter, r *http.Request) {
	c, err := s.Store.GetSharedCoverage(r.Context(), r.PathValue("token"))
	if err != nil {
		s.fail(w, err)
		return
	}
	id := c.ID
	if q := r.URL.Query().Get("id"); q != "" {
		n, err := strconv.ParseInt(q, 10, 64)
		if err != nil || !slices.Contains(c.ViewOverlays, n) {
			s.notFound(w, r)
			return
		}
		id = n
	}
	ra, err := s.Store.CoverageRaster(r.Context(), c.OwnerID, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	png, err := renderRaster(ra)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

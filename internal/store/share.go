package store

import (
	"context"
	"time"
)

// ShareLink enables sharing of the owner's link until expires; an existing token is kept. ErrNotFound when not owned.
func (s *Store) ShareLink(ctx context.Context, owner, id int64, token string, expires time.Time) error {
	return s.share(ctx, `UPDATE links SET share_token = COALESCE(share_token, $3), share_expires_at = $4
		WHERE owner_id = $1 AND id = $2`, owner, id, token, expires)
}

// UnshareLink removes the share token of the owner's link; ErrNotFound when not owned.
func (s *Store) UnshareLink(ctx context.Context, owner, id int64) error {
	return s.unshare(ctx, `UPDATE links SET share_token = NULL, share_expires_at = NULL
		WHERE owner_id = $1 AND id = $2`, owner, id)
}

// GetSharedLink returns the link behind an unexpired share token; ErrNotFound otherwise.
func (s *Store) GetSharedLink(ctx context.Context, token string) (LinkWithSites, error) {
	var l LinkWithSites
	err := scanLink(s.pool.QueryRow(ctx, linkQuery+` WHERE l.share_token = $1 AND l.share_expires_at > now()`, token), &l)
	return l, mapErr(err)
}

// ShareCoverage enables sharing of the owner's coverage until expires; an existing token is kept. ErrNotFound when not owned.
func (s *Store) ShareCoverage(ctx context.Context, owner, id int64, token string, expires time.Time) error {
	return s.share(ctx, `UPDATE coverages SET share_token = COALESCE(share_token, $3), share_expires_at = $4
		WHERE owner_id = $1 AND id = $2`, owner, id, token, expires)
}

// UnshareCoverage removes the share token of the owner's coverage; ErrNotFound when not owned.
func (s *Store) UnshareCoverage(ctx context.Context, owner, id int64) error {
	return s.unshare(ctx, `UPDATE coverages SET share_token = NULL, share_expires_at = NULL
		WHERE owner_id = $1 AND id = $2`, owner, id)
}

// GetSharedCoverage returns the coverage behind an unexpired share token; ErrNotFound otherwise.
func (s *Store) GetSharedCoverage(ctx context.Context, token string) (CoverageWithSite, error) {
	var c CoverageWithSite
	err := scanCoverage(s.pool.QueryRow(ctx, coverageQuery+` WHERE c.share_token = $1 AND c.share_expires_at > now()`, token), &c)
	return c, mapErr(err)
}

// ShareExists reports whether an unexpired share token exists for a link (kind "l") or a coverage ("c").
func (s *Store) ShareExists(ctx context.Context, kind, token string) (bool, error) {
	table := map[string]string{"l": "links", "c": "coverages"}[kind]
	if table == "" {
		return false, nil
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE share_token = $1 AND share_expires_at > now())`, token).Scan(&ok)
	return ok, err
}

// share runs a share UPDATE; ErrNotFound when no row matches.
func (s *Store) share(ctx context.Context, sql string, owner, id int64, token string, expires time.Time) error {
	tag, err := s.pool.Exec(ctx, sql, owner, id, token, expires)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// unshare runs an unshare UPDATE; ErrNotFound when no row matches.
func (s *Store) unshare(ctx context.Context, sql string, owner, id int64) error {
	tag, err := s.pool.Exec(ctx, sql, owner, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

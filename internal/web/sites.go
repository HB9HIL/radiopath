package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/radiopath/radiopath/internal/store"
)

const (
	scopeSitesRead  = "sites:read"
	scopeSitesWrite = "sites:write"
)

type SiteInput struct {
	Name           string  `json:"name"`
	Lat            float64 `json:"lat" doc:"Latitude in degrees, WGS84"`
	Lon            float64 `json:"lon" doc:"Longitude in degrees, WGS84"`
	AntennaHeightM float64 `json:"antenna_height_m" doc:"Antenna height above ground in metres"`
}

type Site struct {
	ID int64 `json:"id"`
	SiteInput
}

type IDPath struct {
	ID int64 `path:"id"`
}

type SiteOutput struct{ Body Site }

// toSite returns the API view of a stored site.
func toSite(st store.Site) Site {
	return Site{ID: st.ID, SiteInput: SiteInput{Name: st.Name, Lat: st.Lat, Lon: st.Lon, AntennaHeightM: st.AntennaHeightM}}
}

// parseSite validates a site input and returns it as a store site of owner.
func parseSite(in SiteInput, owner int64) (store.Site, error) {
	var c check
	st := store.Site{
		OwnerID:        owner,
		Name:           c.text("name", "Name", in.Name),
		Lat:            c.num("lat", "Latitude", in.Lat, -90, 90),
		Lon:            c.num("lon", "Longitude", in.Lon, -180, 180),
		AntennaHeightM: c.num("antenna_height_m", "Antenna height", in.AntennaHeightM, 0.5, 3000),
	}
	return st, c.err()
}

// registerSites registers the site operations.
func (s *Server) registerSites(api huma.API) {
	huma.Register(api, op("list-sites", http.MethodGet, "/api/sites", scopeSitesRead, "The user's sites, ordered by name"),
		func(ctx context.Context, _ *struct{}) (*struct{ Body []Site }, error) {
			sites, err := s.Store.ListSites(ctx, userOf(ctx).ID)
			if err != nil {
				return nil, s.storeErr(err)
			}
			out := make([]Site, 0, len(sites))
			for _, st := range sites {
				out = append(out, toSite(st))
			}
			return &struct{ Body []Site }{out}, nil
		})

	o := op("create-site", http.MethodPost, "/api/sites", scopeSitesWrite, "Create a site")
	o.DefaultStatus = http.StatusCreated
	huma.Register(api, o, func(ctx context.Context, in *struct{ Body SiteInput }) (*SiteOutput, error) {
		st, err := parseSite(in.Body, userOf(ctx).ID)
		if err != nil {
			return nil, err
		}
		st.ID, err = s.Store.CreateSite(ctx, st)
		if errors.Is(err, store.ErrQuota) {
			return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("Limit of %d sites reached", store.MaxSitesPerUser))
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		return &SiteOutput{toSite(st)}, nil
	})

	huma.Register(api, op("get-site", http.MethodGet, "/api/sites/{id}", scopeSitesRead, "One site"),
		func(ctx context.Context, in *IDPath) (*SiteOutput, error) {
			st, err := s.Store.GetSite(ctx, userOf(ctx).ID, in.ID)
			if err != nil {
				return nil, s.storeErr(err)
			}
			return &SiteOutput{toSite(st)}, nil
		})

	huma.Register(api, op("update-site", http.MethodPut, "/api/sites/{id}", scopeSitesWrite, "Replace a site"),
		func(ctx context.Context, in *struct {
			IDPath
			Body SiteInput
		}) (*SiteOutput, error) {
			st, err := parseSite(in.Body, userOf(ctx).ID)
			if err != nil {
				return nil, err
			}
			st.ID = in.ID
			if err := s.Store.UpdateSite(ctx, st); err != nil {
				return nil, s.storeErr(err)
			}
			return &SiteOutput{toSite(st)}, nil
		})

	o = op("delete-site", http.MethodDelete, "/api/sites/{id}", scopeSitesWrite, "Delete a site")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = append(o.Errors, http.StatusConflict)
	o.Description = "409 while a link or coverage uses the site."
	huma.Register(api, o, func(ctx context.Context, in *IDPath) (*struct{}, error) {
		err := s.Store.DeleteSite(ctx, userOf(ctx).ID, in.ID)
		if errors.Is(err, store.ErrInUse) {
			return nil, huma.Error409Conflict("Site is used by a link or coverage and cannot be deleted.")
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		return nil, nil
	})
}

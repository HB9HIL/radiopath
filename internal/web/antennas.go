package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/radiopath/radiopath/internal/antenna"
	"github.com/radiopath/radiopath/internal/plot"
	"github.com/radiopath/radiopath/internal/store"
)

const maxPatternBytes = 1 << 20

// init keeps antenna uploads in memory; MaxBodyBytes of the import bounds them.
func init() {
	humago.MultipartMaxMemory = 2 * maxPatternBytes
}

const (
	scopeAntennasRead  = "antennas:read"
	scopeAntennasWrite = "antennas:write"
	manualSource       = "manual"
)

type ManualAntenna struct {
	BeamwidthDeg  float64  `json:"beamwidth_deg" doc:"Horizontal 3 dB beamwidth, 360 gives an omni"`
	FrontBackDB   float64  `json:"front_back_db" doc:"Front-to-back ratio, also the floor of the pattern; 0 means none"`
	VBeamwidthDeg *float64 `json:"v_beamwidth_deg,omitempty" doc:"Vertical 3 dB beamwidth; without it the elevation cut mirrors the azimuth cut"`
}

type Antenna struct {
	ID           int64          `json:"id"`
	Name         string         `json:"name"`
	GainDBi      float64        `json:"gain_dbi"`
	FreqMHz      *float64       `json:"freq_mhz" nullable:"true"`
	Source       string         `json:"source" doc:"File name of an imported pattern, or manual"`
	FrontDB      float64        `json:"front_db" doc:"Front-to-back ratio of the horizontal pattern"`
	PatternSVG   string         `json:"pattern_svg" doc:"Azimuth pattern as SVG, styled by the classes of the SPA"`
	ElevationSVG *string        `json:"elevation_svg" nullable:"true" doc:"Elevation pattern as SVG; null when the file has none"`
	Manual       *ManualAntenna `json:"manual,omitempty" doc:"The parameters of an antenna described by hand; missing for imported ones"`
}

type ManualAntennaInput struct {
	Name    string   `json:"name"`
	GainDBi float64  `json:"gain_dbi"`
	FreqMHz *float64 `json:"freq_mhz,omitempty"`
	ManualAntenna
}

type AntennaOutput struct{ Body Antenna }

// toAntenna returns the API view of a stored antenna with its pattern plots; manual antennas also carry their parameters.
func toAntenna(a store.Antenna) Antenna {
	p := antenna.Pattern(a.PatternH)
	out := Antenna{ID: a.ID, Name: a.Name, GainDBi: a.GainDBi, FreqMHz: nonZero(a.FreqMHz), Source: a.Source,
		FrontDB: p.Front(), PatternSVG: string(plot.PatternSVG(p, 96))}
	if a.PatternV != nil {
		svg := string(plot.ElevationSVG(a.PatternV, 96))
		out.ElevationSVG = &svg
	}
	if a.Source == manualSource {
		beam, fb := antenna.SectorParams(a.PatternH)
		out.Manual = &ManualAntenna{BeamwidthDeg: beam, FrontBackDB: fb}
		if a.PatternV != nil {
			vbeam, _ := antenna.SectorParams(a.PatternV)
			out.Manual.VBeamwidthDeg = &vbeam
		}
	}
	return out
}

// parseManualAntenna validates a manual antenna and builds sector patterns from beamwidth and front-to-back ratio.
func parseManualAntenna(in ManualAntennaInput, owner int64) (store.Antenna, error) {
	var c check
	rec := store.Antenna{
		Name:    c.text("name", "Name", in.Name),
		GainDBi: c.num("gain_dbi", "Gain", in.GainDBi, -50, 100),
		Source:  manualSource,
		OwnerID: owner,
	}
	if f := c.optNum("freq_mhz", "Frequency", in.FreqMHz, 0, 20000); f != nil {
		rec.FreqMHz = *f
	}
	beam := c.num("beamwidth_deg", "Beamwidth", in.BeamwidthDeg, 1, 360)
	fb := c.num("front_back_db", "Front-to-back", in.FrontBackDB, 0, 60)
	rec.PatternH = antenna.Sector(beam, fb)
	if vbeam := c.optNum("v_beamwidth_deg", "Vertical beamwidth", in.VBeamwidthDeg, 1, 360); vbeam != nil {
		rec.PatternV = antenna.Sector(*vbeam, fb)
	}
	return rec, c.err()
}

// registerAntennas registers the antenna operations: list, import of MSI files, manual sectors, edit and delete.
func (s *Server) registerAntennas(api huma.API) {
	huma.Register(api, op("list-antennas", http.MethodGet, "/api/antennas", scopeAntennasRead, "The user's antennas"),
		func(ctx context.Context, _ *struct{}) (*struct{ Body []Antenna }, error) {
			as, err := s.Store.ListAntennas(ctx, userOf(ctx).ID)
			if err != nil {
				return nil, s.storeErr(err)
			}
			out := make([]Antenna, 0, len(as))
			for _, a := range as {
				out = append(out, toAntenna(a))
			}
			return &struct{ Body []Antenna }{out}, nil
		})

	o := op("import-antenna", http.MethodPost, "/api/antennas/import", scopeAntennasWrite, "Import an MSI Planet pattern file")
	o.DefaultStatus = http.StatusCreated
	o.MaxBodyBytes = 2 * maxPatternBytes
	o.Description = "The name falls back to the one in the file, then to the file name. A gain without a unit is read as dBd."
	huma.Register(api, o, func(ctx context.Context, in *struct {
		RawBody huma.MultipartFormFiles[struct {
			File huma.FormFile `form:"file" contentType:"application/octet-stream,text/plain" required:"true"`
			Name string        `form:"name" required:"false"`
		}]
	}) (*AntennaOutput, error) {
		data := in.RawBody.Data()
		a, err := antenna.ParseMSI(data.File)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("Invalid input", &huma.ErrorDetail{Location: "body.file", Message: err.Error()})
		}
		name := strings.TrimSpace(data.Name)
		if name == "" {
			name = strings.TrimSpace(a.Name)
		}
		if name == "" {
			name = data.File.Filename
		}
		rec := store.Antenna{
			Name: name, GainDBi: a.GainDBi, FreqMHz: a.FreqMHz, PatternH: a.Horizontal, PatternV: a.Vertical,
			Source: data.File.Filename, OwnerID: userOf(ctx).ID,
		}
		if rec.ID, err = s.Store.CreateAntenna(ctx, rec); err != nil {
			return nil, s.storeErr(err)
		}
		return &AntennaOutput{toAntenna(rec)}, nil
	})

	o = op("create-antenna", http.MethodPost, "/api/antennas", scopeAntennasWrite, "Describe a sector antenna by hand")
	o.DefaultStatus = http.StatusCreated
	o.Description = "Builds the usual sector shape: 12 (angle / beamwidth)² dB, capped at the front-to-back ratio."
	huma.Register(api, o, func(ctx context.Context, in *struct{ Body ManualAntennaInput }) (*AntennaOutput, error) {
		rec, err := parseManualAntenna(in.Body, userOf(ctx).ID)
		if err != nil {
			return nil, err
		}
		if rec.ID, err = s.Store.CreateAntenna(ctx, rec); err != nil {
			return nil, s.storeErr(err)
		}
		return &AntennaOutput{toAntenna(rec)}, nil
	})

	o = op("update-antenna", http.MethodPut, "/api/antennas/{id}", scopeAntennasWrite, "Replace a manual antenna")
	o.Errors = append(o.Errors, http.StatusConflict)
	o.Description = "Links and coverages using the antenna take over the new gain and pattern. 409 for imported antennas."
	huma.Register(api, o, func(ctx context.Context, in *struct {
		IDPath
		Body ManualAntennaInput
	}) (*AntennaOutput, error) {
		owner := userOf(ctx).ID
		a, err := s.Store.GetAntenna(ctx, owner, in.ID)
		if err != nil {
			return nil, s.storeErr(err)
		}
		if a.Source != manualSource {
			return nil, huma.Error409Conflict("Imported antennas cannot be edited")
		}
		rec, err := parseManualAntenna(in.Body, owner)
		if err != nil {
			return nil, err
		}
		rec.ID = a.ID
		if err := s.Store.UpdateAntenna(ctx, rec); err != nil {
			return nil, s.storeErr(err)
		}
		return &AntennaOutput{toAntenna(rec)}, nil
	})

	o = op("delete-antenna", http.MethodDelete, "/api/antennas/{id}", scopeAntennasWrite, "Delete an antenna")
	o.DefaultStatus = http.StatusNoContent
	o.Description = "Links and coverages using it become omnidirectional."
	huma.Register(api, o, func(ctx context.Context, in *IDPath) (*struct{}, error) {
		if err := s.Store.DeleteAntenna(ctx, userOf(ctx).ID, in.ID); err != nil {
			return nil, s.storeErr(err)
		}
		return nil, nil
	})
}

// nonZero returns a pointer to v, or nil when v is zero.
func nonZero(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

// pattern loads the antenna's radiation pattern; it returns an empty (omnidirectional) lobe when id is nil or the lookup fails.
func (s *Server) pattern(ctx context.Context, owner int64, id *int64) antenna.Lobe {
	if id == nil {
		return antenna.Lobe{}
	}
	a, err := s.Store.GetAntenna(ctx, owner, *id)
	if err != nil {
		s.Log.Warn("antenna pattern not loaded", "antenna", *id, "err", err)
		return antenna.Lobe{}
	}
	return antenna.Lobe{H: a.PatternH, V: a.PatternV}
}

// azimuth returns *deg, or fallback when deg is nil.
func azimuth(deg *float64, fallback float64) float64 {
	if deg == nil {
		return fallback
	}
	return *deg
}

// antennaGain returns the selected antenna's gain, or the entered gain when none is selected or it cannot be loaded.
func (s *Server) antennaGain(ctx context.Context, owner int64, id *int64, entered float64) float64 {
	if id == nil {
		return entered
	}
	a, err := s.Store.GetAntenna(ctx, owner, *id)
	if err != nil {
		return entered
	}
	return a.GainDBi
}

// antennaName returns the name of the owner's antenna, or an empty string when id is nil or the lookup fails.
func (s *Server) antennaName(ctx context.Context, owner int64, id *int64) string {
	if id == nil {
		return ""
	}
	a, err := s.Store.GetAntenna(ctx, owner, *id)
	if err != nil {
		return ""
	}
	return a.Name
}

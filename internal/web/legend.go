package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/radiopath/radiopath/internal/plot"
	"github.com/radiopath/radiopath/internal/store"
)

type LegendBin struct {
	MinDB float64 `json:"min_db" doc:"Lowest link margin of the bin"`
	Color string  `json:"color" doc:"Colour as #rrggbb"`
}

type LegendRow struct {
	LegendBin
	Label string `json:"label" doc:"The margin range as text"`
}

// legendRows returns the bins of l with their labels.
func legendRows(l plot.Legend) []LegendRow {
	out := make([]LegendRow, len(l))
	for i, b := range l {
		out[i] = LegendRow{LegendBin{b.MinDB, b.Hex()}, l.Label(i)}
	}
	return out
}

// newLegend validates bins as a legend, sorted from high to low threshold.
func newLegend(bins []LegendBin) (plot.Legend, error) {
	mins, colors := make([]float64, len(bins)), make([]string, len(bins))
	for i, b := range bins {
		mins[i], colors[i] = b.MinDB, b.Color
	}
	return plot.NewLegend(mins, colors)
}

// legendOf returns the coverage's custom legend, or the default when it has none.
func legendOf(c store.Coverage) plot.Legend { return plot.LegendOr(c.LegendDB, c.LegendColors) }

// registerLegend registers PUT /api/coverages/{id}/legend.
func (s *Server) registerLegend(api huma.API) {
	o := op("set-coverage-legend", http.MethodPut, "/api/coverages/{id}/legend", scopeCoveragesWrite, "Set the colours of a coverage")
	o.DefaultStatus = http.StatusNoContent
	o.Description = fmt.Sprintf("A cell takes the colour of the highest threshold its margin reaches. 1 to %d bins; none resets to the default. Nothing is recomputed.", plot.MaxBins)
	huma.Register(api, o, func(ctx context.Context, in *struct {
		IDPath
		Body struct {
			Bins []LegendBin `json:"bins"`
		}
	}) (*struct{}, error) {
		var mins []float64
		var colors []string
		if len(in.Body.Bins) > 0 {
			l, err := newLegend(in.Body.Bins)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("Invalid input", &huma.ErrorDetail{Location: "body.bins", Message: err.Error()})
			}
			mins, colors = l.Mins(), l.Hex()
		}
		if err := s.Store.SetCoverageLegend(ctx, userOf(ctx).ID, in.ID, mins, colors); err != nil {
			return nil, s.storeErr(err)
		}
		return nil, nil
	})
}

// renderRaster returns the stored PNG, or re-renders it from the margin data with the current legend when one is stored.
func renderRaster(ra store.Raster) ([]byte, error) {
	if ra.Margin == nil {
		return ra.PNG, nil
	}
	res, err := plot.DecodeRaster(ra.Margin, ra.Bounds.North, ra.Bounds.South, ra.Bounds.East, ra.Bounds.West)
	if err != nil {
		return nil, err
	}
	return plot.CoveragePNG(res, plot.LegendOr(ra.LegendDB, ra.LegendColors)), nil
}

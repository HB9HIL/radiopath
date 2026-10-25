package itm

import "math"

// freeSpaceLoss returns the free-space path loss in dB for distance d (m) and frequency f (MHz).
func freeSpaceLoss(d__meter, f__mhz float64) float64 {
	return 32.45 + 20.0*math.Log10(f__mhz) + 20.0*math.Log10(d__meter/1000.0)
}

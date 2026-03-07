package cache

import "math"

// AngleCache stores pre-computed trigonometric values for cube-to-sphere coordinate mapping
// in flat arrays for optimal CPU cache locality. Index with [y*Size + x].
type AngleCache struct {
	Size      int
	ZenithPos []float64 // acos(1/r) — zenith angle for upward-facing tile
	ZenithNeg []float64 // acos(-1/r) — zenith angle for downward-facing tile
	Polar     []float64 // acos(y/r) — polar angle for side tiles
	Azimuth   []float64 // atan(y/x) — azimuth angle
}

// At returns the flat index for (x, y) coordinates.
func (c *AngleCache) At(x, y int) int {
	return y*c.Size + x
}

// Precompute builds lookup tables for all trigonometric values needed
// to map tile pixel coordinates to spherical coordinates.
func Precompute(tileSize int) AngleCache {
	n := tileSize * tileSize
	halfSize := float64(tileSize-1) / 2

	c := AngleCache{
		Size:      tileSize,
		ZenithPos: make([]float64, n),
		ZenithNeg: make([]float64, n),
		Polar:     make([]float64, n),
		Azimuth:   make([]float64, n),
	}

	for tileY := range tileSize {
		y := float64(tileY)/halfSize - 1
		rowOff := tileY * tileSize

		for tileX := range tileSize {
			x := float64(tileX)/halfSize - 1
			root := math.Sqrt(x*x + y*y + 1)
			i := rowOff + tileX
			c.ZenithPos[i] = math.Acos(1 / root)
			c.ZenithNeg[i] = math.Acos(-1 / root)
			c.Polar[i] = math.Acos(y / root)
			if x != 0 {
				c.Azimuth[i] = math.Atan(y / x)
			}
		}
	}

	return c
}

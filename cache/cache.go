package cache

import "math"

// AngleCache stores pre-computed trigonometric values for cube-to-sphere coordinate mapping.
// Pre-computing these avoids redundant math operations during per-pixel processing.
type AngleCache struct {
	ZenithPos  [][]float64 // acos(1/r) — zenith angle for upward-facing tile
	ZenithNeg  [][]float64 // acos(-1/r) — zenith angle for downward-facing tile
	Polar      [][]float64 // acos(y/r) — polar angle for side tiles
	Azimuth    [][]float64 // atan(y/x) — azimuth angle
}

// Precompute builds lookup tables for all trigonometric values needed
// to map tile pixel coordinates to spherical coordinates.
func Precompute(tileSize int) AngleCache {
	halfSize := float64(tileSize-1) / 2

	c := AngleCache{
		ZenithPos: make([][]float64, tileSize),
		ZenithNeg: make([][]float64, tileSize),
		Polar:     make([][]float64, tileSize),
		Azimuth:   make([][]float64, tileSize),
	}

	for tileY := range tileSize {
		y := float64(tileY)/halfSize - 1

		c.ZenithPos[tileY] = make([]float64, tileSize)
		c.ZenithNeg[tileY] = make([]float64, tileSize)
		c.Polar[tileY] = make([]float64, tileSize)
		c.Azimuth[tileY] = make([]float64, tileSize)

		for tileX := range tileSize {
			x := float64(tileX)/halfSize - 1
			root := math.Sqrt(x*x + y*y + 1)
			c.ZenithPos[tileY][tileX] = math.Acos(1 / root)
			c.ZenithNeg[tileY][tileX] = math.Acos(-1 / root)
			c.Polar[tileY][tileX] = math.Acos(y / root)
			if x != 0 {
				c.Azimuth[tileY][tileX] = math.Atan(y / x)
			}
		}
	}

	return c
}

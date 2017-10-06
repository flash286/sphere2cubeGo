package cache

import "math"

type CacheAngles struct {
	ZP   [][]float64
	ZM   [][]float64
	XYPM [][]float64
	PHI  [][]float64
}

func CacheAnglesHandler(tileSize int) CacheAngles {
	var cache CacheAngles
	half_size := float64(tileSize-1) / 2

	cache.ZP = make([][]float64, tileSize)
	cache.ZM = make([][]float64, tileSize)
	cache.XYPM = make([][]float64, tileSize)
	cache.PHI = make([][]float64, tileSize)

	for tileY := 0; tileY < tileSize; tileY++ {
		y := float64(tileY)/half_size - 1

		if cache.ZP[tileY] == nil {
			cache.ZP[tileY] = make([]float64, tileSize)
		}
		if cache.ZM[tileY] == nil {
			cache.ZM[tileY] = make([]float64, tileSize)
		}
		if cache.XYPM[tileY] == nil {
			cache.XYPM[tileY] = make([]float64, tileSize)
		}
		if cache.PHI[tileY] == nil {
			cache.PHI[tileY] = make([]float64, tileSize)
		}

		for tileX := 0; tileX < tileSize; tileX++ {
			x := float64(tileX)/half_size - 1
			root := math.Sqrt(x*x + y*y + 1)
			cache.ZP[tileY][tileX] = math.Acos(1 / root)
			cache.ZM[tileY][tileX] = math.Acos(-1 / root)
			cache.XYPM[tileY][tileX] = math.Acos(float64(y) / root)
			if x != 0 {
				cache.PHI[tileY][tileX] = math.Atan(float64(y) / x)
			}
		}
	}

	return cache
}

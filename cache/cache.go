package cache

import (
	"math"
)

type CacheAngles struct {
	ZP   [][]float64
	ZM   [][]float64
	XYPM [][]float64
	PHI  [][]float64
}

func CacheAnglesHandler(tileSize int) CacheAngles {
	var cache CacheAngles
	halfSize := float64(tileSize-1) / 2

	cache.ZP = make([][]float64, tileSize)
	cache.ZM = make([][]float64, tileSize)
	cache.XYPM = make([][]float64, tileSize)
	cache.PHI = make([][]float64, tileSize)

	for tileY := 0; tileY < tileSize; tileY++ {
		y := float64(tileY)/halfSize - 1

		cache.ZP[tileY] = make([]float64, tileSize)
		cache.ZM[tileY] = make([]float64, tileSize)
		cache.XYPM[tileY] = make([]float64, tileSize)
		cache.PHI[tileY] = make([]float64, tileSize)

		for tileX := 0; tileX < tileSize; tileX++ {
			x := float64(tileX)/halfSize - 1
			root := math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2) + 1)
			cache.ZP[tileY][tileX] = math.Acos(1 / root)
			cache.ZM[tileY][tileX] = math.Acos(-1 / root)
			cache.XYPM[tileY][tileX] = math.Acos(y / root)
			if x != 0 {
				cache.PHI[tileY][tileX] = math.Atan(y / x)
			}
		}
	}

	return cache
}
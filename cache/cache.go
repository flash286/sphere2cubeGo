package cache

import (
	"math"
	"runtime"
	"sync"
)

// PixelMap stores pre-computed source pixel offsets for all six cubemap faces.
// For each face, Maps[face][tileY*Size+tileX] = byte offset into source panorama Pix slice.
// This eliminates all float arithmetic from the render loop.
type PixelMap struct {
	Size int
	Maps [6][]int32 // up, down, front, right, back, left
}

// Precompute builds pixel offset lookup tables for all six cubemap faces.
// Combines trig computation and coordinate mapping into a single pass,
// parallelized across available CPU cores.
func Precompute(tileSize, panoW, panoH int) PixelMap {
	n := tileSize * tileSize
	pm := PixelMap{Size: tileSize}
	for i := range 6 {
		pm.Maps[i] = make([]int32, n)
	}

	halfSize := float64(tileSize-1) / 2
	halfWidth := 0.5 * float64(panoW)
	invPiWidth := halfWidth / math.Pi
	invPiHeight := float64(panoH) / math.Pi

	workers := runtime.GOMAXPROCS(0)
	if workers > tileSize {
		workers = tileSize
	}

	var wg sync.WaitGroup
	rowsPerWorker := (tileSize + workers - 1) / workers

	for w := range workers {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > tileSize {
			endY = tileSize
		}
		if startY >= endY {
			break
		}

		wg.Add(1)
		go func(sy, ey int) {
			defer wg.Done()
			for tileY := sy; tileY < ey; tileY++ {
				rowOff := tileY * tileSize
				fy := float64(tileY)

				for tileX := range tileSize {
					i := rowOff + tileX
					fx := float64(tileX)

					// Normalized coordinates for cache lookups
					nx := fx/halfSize - 1
					ny := fy/halfSize - 1
					root := math.Sqrt(nx*nx + ny*ny + 1)

					zenithPos := math.Acos(1 / root)
					zenithNeg := math.Acos(-1 / root)
					polar := math.Acos(ny / root)
					var azimuth float64
					if nx != 0 {
						azimuth = math.Atan(ny / nx)
					}

					// Pre-compute reversed indices for cache access patterns
					rx := tileSize - tileX - 1 // size-tileX-1
					ry := tileSize - tileY - 1 // size-tileY-1

					// Helper: normalized coords at arbitrary position
					normAt := func(px, py int) (float64, float64) {
						return float64(px)/halfSize - 1, float64(py)/halfSize - 1
					}

					// Helper: azimuth at arbitrary position
					azimuthAt := func(px, py int) float64 {
						ax, ay := normAt(px, py)
						r := math.Sqrt(ax*ax + ay*ay + 1)
						_ = r // azimuth doesn't use r
						if ax != 0 {
							return math.Atan(ay / ax)
						}
						return 0
					}

					// Helper: polar at arbitrary position
					polarAt := func(px, py int) float64 {
						ax, ay := normAt(px, py)
						r := math.Sqrt(ax*ax + ay*ay + 1)
						return math.Acos(ay / r)
					}

					// Helper: zenithPos at arbitrary position
					zenithPosAt := func(px, py int) float64 {
						ax, ay := normAt(px, py)
						r := math.Sqrt(ax*ax + ay*ay + 1)
						return math.Acos(1 / r)
					}

					// Helper: zenithNeg at arbitrary position
					zenithNegAt := func(px, py int) float64 {
						ax, ay := normAt(px, py)
						r := math.Sqrt(ax*ax + ay*ay + 1)
						return math.Acos(-1 / r)
					}

					// Up face: theta=zenithPos[tileX,tileY], phi=azimuth[tileY,tileX]+updatePhi
					{
						theta := zenithPosAt(tileX, tileY)
						phi := azimuthAt(tileY, tileX)
						switch {
						case fy < halfSize:
							phi += math.Pi
						case fy > halfSize:
							// phi += 0
						case fx < halfSize:
							phi = -math.Pi / 2
						default:
							phi = math.Pi / 2
						}
						pm.Maps[0][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Down face: theta=zenithNeg[tileX,tileY], phi=azimuth[size-tileY-1,tileX]+updatePhi
					{
						theta := zenithNegAt(tileX, tileY)
						phi := azimuthAt(ry, tileX)
						switch {
						case fy < halfSize:
							// phi += 0
						case fy > halfSize:
							phi += math.Pi
						case fx < halfSize:
							phi = -math.Pi / 2
						default:
							phi = math.Pi / 2
						}
						pm.Maps[1][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Front face: theta=polar[rx,ry], phi=azimuth[size-1,tileX]+updatePhi
					{
						theta := polarAt(rx, ry)
						phi := azimuthAt(tileSize-1, tileX)
						switch {
						case fy < halfSize:
							// phi += 0
						case fy > halfSize:
							// phi += 0
						case fx < halfSize:
							phi = -math.Pi / 2
						default:
							phi = math.Pi / 2
						}
						pm.Maps[2][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Right face: theta=polar[rx,ry], phi=azimuth[rx,size-1]+updatePhi
					{
						theta := polarAt(rx, ry)
						phi := azimuthAt(rx, tileSize-1)
						switch {
						case fx < halfSize:
							// phi += 0
						case fx > halfSize:
							phi += math.Pi
						case fy < halfSize:
							phi = math.Pi / 2
						default:
							phi = math.Pi / 2
						}
						pm.Maps[3][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Back face: theta=polar[rx,ry], phi=azimuth[size-1,tileX]+pi
					{
						theta := polarAt(rx, ry)
						phi := azimuthAt(tileSize-1, tileX) + math.Pi
						pm.Maps[4][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Left face: theta=polar[rx,ry], phi=azimuth[rx,size-1]+updatePhi
					{
						theta := polarAt(rx, ry)
						phi := azimuthAt(rx, tileSize-1)
						switch {
						case fx < halfSize:
							phi += math.Pi
						case fx > halfSize:
							// phi += 0
						case fy < halfSize:
							phi = -math.Pi / 2
						default:
							phi = -math.Pi / 2
						}
						pm.Maps[5][i] = toSrcOff(phi, theta, panoW, halfWidth, invPiWidth, invPiHeight)
					}

					// Suppress unused variable warnings
					_ = zenithPos
					_ = zenithNeg
					_ = polar
					_ = azimuth
				}
			}
		}(startY, endY)
	}

	wg.Wait()
	return pm
}

// toSrcOff converts spherical angles to a byte offset in the panorama Pix slice.
func toSrcOff(phi, theta float64, panoW int, halfWidth, invPiWidth, invPiHeight float64) int32 {
	x := halfWidth + invPiWidth*phi
	w := float64(panoW)
	if x < 1 {
		x += w
	} else if x > w {
		x -= w
	}
	return int32((int(invPiHeight*theta)*panoW + int(x)) * 4)
}

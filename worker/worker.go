package worker

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"sphere2cubeGo/cache"
)

// Face names for cubemap tiles.
const (
	FaceUp    = "up"
	FaceDown  = "down"
	FaceFront = "front"
	FaceRight = "right"
	FaceBack  = "back"
	FaceLeft  = "left"
)

// AllFaces returns all six cubemap face names.
func AllFaces() []string {
	return []string{FaceUp, FaceDown, FaceFront, FaceRight, FaceBack, FaceLeft}
}

// Tile describes a single cubemap face to generate.
type Tile struct {
	Size int
	Name string
}

func (t *Tile) halfSize() float64 {
	return float64(t.Size-1) / 2
}

// TileResult holds the generated image for a cubemap face.
type TileResult struct {
	Tile  Tile
	Image *image.NRGBA
}

// Panorama stores equirectangular image pixels as a flat NRGBA byte slice.
// 4 bytes per pixel (R, G, B, A), row-major order.
type Panorama struct {
	Width  int
	Height int
	Pix    []uint8 // len = Width * Height * 4
}

// pixOffset returns the byte offset for pixel at (x, y).
func (p *Panorama) pixOffset(x, y int) int {
	return (y*p.Width + x) * 4
}

// ValidateAspectRatio checks that the panorama has a 2:1 width-to-height ratio.
func (p *Panorama) ValidateAspectRatio() error {
	if p.Width == 0 || p.Height == 0 {
		return fmt.Errorf("empty image")
	}
	if p.Width/p.Height != 2 {
		return fmt.Errorf("panorama must have 2:1 aspect ratio, got %d:%d", p.Width, p.Height)
	}
	return nil
}

// DecodeImage reads an image (JPEG or PNG) and returns a Panorama with flat pixel data.
func DecodeImage(r io.Reader) (*Panorama, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Fast path: if already NRGBA, just take the Pix slice directly.
	if nrgba, ok := img.(*image.NRGBA); ok && nrgba.Stride == w*4 {
		return &Panorama{Width: w, Height: h, Pix: nrgba.Pix}, nil
	}

	pix := make([]uint8, w*h*4)
	i := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pix[i] = uint8(r >> 8)
			pix[i+1] = uint8(g >> 8)
			pix[i+2] = uint8(b >> 8)
			pix[i+3] = uint8(a >> 8)
			i += 4
		}
	}

	return &Panorama{Width: w, Height: h, Pix: pix}, nil
}

// NewPanorama creates a Panorama from raw NRGBA pixel data.
// Useful for testing without JPEG encode/decode overhead.
func NewPanorama(width, height int, pix []uint8) *Panorama {
	return &Panorama{Width: width, Height: height, Pix: pix}
}

func updatePhi(halfSize float64, phi float64, majorDir, minorDir int, majorM, majorP, minorM, minorP float64) float64 {
	switch {
	case float64(majorDir) < halfSize:
		return phi + majorM
	case float64(majorDir) > halfSize:
		return phi + majorP
	case float64(minorDir) < halfSize:
		return minorM
	default:
		return minorP
	}
}

func phi2Width(width int, phi float64) int {
	x := 0.5 * float64(width) * (phi/math.Pi + 1)
	w := float64(width)
	if x < 1 {
		x += w
	} else if x > w {
		x -= w
	}
	return int(x)
}

func theta2Height(height int, theta float64) int {
	return int(float64(height) * theta / math.Pi)
}

// RenderTile generates a cubemap face image from the source panorama.
// Writes directly to the output image's Pix slice to avoid per-pixel allocations.
func RenderTile(pano *Panorama, tile Tile, ac cache.AngleCache) TileResult {
	size := tile.Size
	tileImage := image.NewNRGBA(image.Rect(0, 0, size, size))
	dst := tileImage.Pix
	src := pano.Pix
	srcW := pano.Width
	srcH := pano.Height
	half := tile.halfSize()

	for tileY := range size {
		dstRowOff := tileY * size * 4

		for tileX := range size {
			var theta, phi float64

			switch tile.Name {
			case FaceUp:
				theta = ac.ZenithPos[ac.At(tileX, tileY)]
				phi = ac.Azimuth[ac.At(tileY, tileX)]
				phi = updatePhi(half, phi, tileY, tileX, math.Pi, 0, -math.Pi/2, math.Pi/2)
			case FaceDown:
				theta = ac.ZenithNeg[ac.At(tileX, tileY)]
				phi = ac.Azimuth[ac.At(size-tileY-1, tileX)]
				phi = updatePhi(half, phi, tileY, tileX, 0, math.Pi, -math.Pi/2, math.Pi/2)
			case FaceFront:
				theta = ac.Polar[ac.At(size-tileX-1, size-tileY-1)]
				phi = ac.Azimuth[ac.At(size-1, tileX)]
				phi = updatePhi(half, phi, tileY, tileX, 0, 0, -math.Pi/2, math.Pi/2)
			case FaceRight:
				theta = ac.Polar[ac.At(size-tileX-1, size-tileY-1)]
				phi = ac.Azimuth[ac.At(size-tileX-1, size-1)]
				phi = updatePhi(half, phi, tileX, tileY, 0, math.Pi, math.Pi/2, math.Pi/2)
			case FaceBack:
				theta = ac.Polar[ac.At(size-tileX-1, size-tileY-1)]
				phi = ac.Azimuth[ac.At(size-1, tileX)] + math.Pi
			case FaceLeft:
				theta = ac.Polar[ac.At(size-tileX-1, size-tileY-1)]
				phi = ac.Azimuth[ac.At(size-tileX-1, size-1)]
				phi = updatePhi(half, phi, tileX, tileY, math.Pi, 0, -math.Pi/2, -math.Pi/2)
			}

			spX := phi2Width(srcW, phi)
			spY := theta2Height(srcH, theta)

			srcOff := (spY*srcW + spX) * 4
			dstOff := dstRowOff + tileX*4

			dst[dstOff] = src[srcOff]
			dst[dstOff+1] = src[srcOff+1]
			dst[dstOff+2] = src[srcOff+2]
			dst[dstOff+3] = src[srcOff+3]
		}
	}

	return TileResult{Tile: tile, Image: tileImage}
}

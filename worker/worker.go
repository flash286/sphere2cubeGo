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

// TileResult holds the generated image for a cubemap face.
type TileResult struct {
	Tile  Tile
	Image *image.RGBA
}

// Panorama stores equirectangular image pixels as a flat NRGBA byte slice.
// 4 bytes per pixel (R, G, B, A), row-major order.
type Panorama struct {
	Width  int
	Height int
	Pix    []uint8 // len = Width * Height * 4
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

// renderParams holds pre-extracted values to avoid repeated field access in hot loops.
type renderParams struct {
	dst  []uint8
	src  []uint8
	srcW int
	size int
	half float64

	zenithPos []float64
	zenithNeg []float64
	polar     []float64
	azimuth   []float64

	halfWidth   float64
	invPiWidth  float64
	invPiHeight float64
}

func newRenderParams(pano *Panorama, tileImage *image.RGBA, size int, ac *cache.AngleCache) renderParams {
	return renderParams{
		dst:         tileImage.Pix,
		src:         pano.Pix,
		srcW:        pano.Width,
		size:        size,
		half:        float64(size-1) / 2,
		zenithPos:   ac.ZenithPos,
		zenithNeg:   ac.ZenithNeg,
		polar:       ac.Polar,
		azimuth:     ac.Azimuth,
		halfWidth:   0.5 * float64(pano.Width),
		invPiWidth:  0.5 * float64(pano.Width) / math.Pi,
		invPiHeight: float64(pano.Height) / math.Pi,
	}
}

// mapPixel converts spherical angles to panorama coordinates and copies the pixel.
func (p *renderParams) mapPixel(dstOff int, theta, phi float64) {
	x := p.halfWidth + p.invPiWidth*phi
	w := float64(p.srcW)
	if x < 1 {
		x += w
	} else if x > w {
		x -= w
	}

	srcOff := (int(p.invPiHeight*theta)*p.srcW + int(x)) * 4
	copy(p.dst[dstOff:dstOff+4], p.src[srcOff:srcOff+4])
}

func renderUp(p *renderParams) {
	size := p.size
	half := p.half

	for tileY := range size {
		dstRowOff := tileY * size * 4
		fy := float64(tileY)

		for tileX := range size {
			theta := p.zenithPos[tileY*size+tileX]
			phi := p.azimuth[tileX*size+tileY]

			switch {
			case fy < half:
				phi += math.Pi
			case fy > half:
				// phi += 0
			case float64(tileX) < half:
				phi = -math.Pi / 2
			default:
				phi = math.Pi / 2
			}

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

func renderDown(p *renderParams) {
	size := p.size
	half := p.half

	for tileY := range size {
		dstRowOff := tileY * size * 4
		fy := float64(tileY)

		for tileX := range size {
			theta := p.zenithNeg[tileY*size+tileX]
			phi := p.azimuth[(size-tileY-1)*size+tileX]

			switch {
			case fy < half:
				// phi += 0
			case fy > half:
				phi += math.Pi
			case float64(tileX) < half:
				phi = -math.Pi / 2
			default:
				phi = math.Pi / 2
			}

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

func renderFront(p *renderParams) {
	size := p.size
	half := p.half

	for tileY := range size {
		dstRowOff := tileY * size * 4
		fy := float64(tileY)

		for tileX := range size {
			idx := (size-tileX-1)*size + (size - tileY - 1)
			theta := p.polar[idx]
			phi := p.azimuth[(size-1)*size+tileX]

			switch {
			case fy < half:
				// phi += 0
			case fy > half:
				// phi += 0
			case float64(tileX) < half:
				phi = -math.Pi / 2
			default:
				phi = math.Pi / 2
			}

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

func renderRight(p *renderParams) {
	size := p.size
	half := p.half

	for tileY := range size {
		dstRowOff := tileY * size * 4

		for tileX := range size {
			idx := (size-tileX-1)*size + (size - tileY - 1)
			theta := p.polar[idx]
			phi := p.azimuth[(size-tileX-1)*size+(size-1)]
			fx := float64(tileX)

			switch {
			case fx < half:
				// phi += 0
			case fx > half:
				phi += math.Pi
			case float64(tileY) < half:
				phi = math.Pi / 2
			default:
				phi = math.Pi / 2
			}

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

func renderBack(p *renderParams) {
	size := p.size

	for tileY := range size {
		dstRowOff := tileY * size * 4

		for tileX := range size {
			idx := (size-tileX-1)*size + (size - tileY - 1)
			theta := p.polar[idx]
			phi := p.azimuth[(size-1)*size+tileX] + math.Pi

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

func renderLeft(p *renderParams) {
	size := p.size
	half := p.half

	for tileY := range size {
		dstRowOff := tileY * size * 4

		for tileX := range size {
			idx := (size-tileX-1)*size + (size - tileY - 1)
			theta := p.polar[idx]
			phi := p.azimuth[(size-tileX-1)*size+(size-1)]
			fx := float64(tileX)

			switch {
			case fx < half:
				phi += math.Pi
			case fx > half:
				// phi += 0
			case float64(tileY) < half:
				phi = -math.Pi / 2
			default:
				phi = -math.Pi / 2
			}

			p.mapPixel(dstRowOff+tileX*4, theta, phi)
		}
	}
}

// RenderTile generates a cubemap face image from the source panorama.
// Dispatches to face-specific render functions to avoid per-pixel branching.
func RenderTile(pano *Panorama, tile Tile, ac *cache.AngleCache) TileResult {
	size := tile.Size
	tileImage := image.NewRGBA(image.Rect(0, 0, size, size))
	p := newRenderParams(pano, tileImage, size, ac)

	switch tile.Name {
	case FaceUp:
		renderUp(&p)
	case FaceDown:
		renderDown(&p)
	case FaceFront:
		renderFront(&p)
	case FaceRight:
		renderRight(&p)
	case FaceBack:
		renderBack(&p)
	case FaceLeft:
		renderLeft(&p)
	}

	return TileResult{Tile: tile, Image: tileImage}
}

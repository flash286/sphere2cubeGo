package worker

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"sphere2cubeGo/cache"
)

// Face indices matching cache.PixelMap.Maps order.
const (
	faceIdxUp    = 0
	faceIdxDown  = 1
	faceIdxFront = 2
	faceIdxRight = 3
	faceIdxBack  = 4
	faceIdxLeft  = 5
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

var faceIndex = map[string]int{
	FaceUp:    faceIdxUp,
	FaceDown:  faceIdxDown,
	FaceFront: faceIdxFront,
	FaceRight: faceIdxRight,
	FaceBack:  faceIdxBack,
	FaceLeft:  faceIdxLeft,
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

// RenderTile generates a cubemap face image from the source panorama.
// Uses pre-computed pixel offset map for zero float math in the hot loop.
func RenderTile(pano *Panorama, tile Tile, pm *cache.PixelMap) TileResult {
	size := tile.Size
	tileImage := image.NewRGBA(image.Rect(0, 0, size, size))
	dst := tileImage.Pix
	src := pano.Pix
	offsets := pm.Maps[faceIndex[tile.Name]]

	n := size * size
	for i := range n {
		srcOff := int(offsets[i])
		dstOff := i * 4
		a := src[srcOff+3]
		switch a {
		case 255:
			copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		case 0:
			// Fully transparent — leave dst as zero (already zeroed by NewRGBA).
		default:
			a16 := uint16(a)
			dst[dstOff] = uint8((uint16(src[srcOff])*a16 + 127) / 255)
			dst[dstOff+1] = uint8((uint16(src[srcOff+1])*a16 + 127) / 255)
			dst[dstOff+2] = uint8((uint16(src[srcOff+2])*a16 + 127) / 255)
			dst[dstOff+3] = a
		}
	}

	return TileResult{Tile: tile, Image: tileImage}
}

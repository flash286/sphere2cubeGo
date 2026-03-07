package worker

import (
	"fmt"
	"image"
	"image/color"
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
	Image image.Image
}

// Pixel holds 8-bit RGBA color values.
type Pixel struct {
	R, G, B, A int
}

func (p *Pixel) toColor() color.Color {
	return color.RGBA64{
		uint16(p.R * 257),
		uint16(p.G * 257),
		uint16(p.B * 257),
		uint16(p.A * 257),
	}
}

func rgbaToPixel(r, g, b, a uint32) Pixel {
	return Pixel{int(r / 257), int(g / 257), int(b / 257), int(a / 257)}
}

// DecodePixels reads an image (JPEG or PNG) and returns a 2D pixel array.
func DecodePixels(r io.Reader) ([][]Pixel, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	pixels := make([][]Pixel, height)
	for y := range height {
		row := make([]Pixel, width)
		for x := range width {
			row[x] = rgbaToPixel(img.At(x, y).RGBA())
		}
		pixels[y] = row
	}

	return pixels, nil
}

// ValidateAspectRatio checks that the panorama has a 2:1 width-to-height ratio.
func ValidateAspectRatio(pixels [][]Pixel) error {
	if len(pixels) == 0 || len(pixels[0]) == 0 {
		return fmt.Errorf("empty image")
	}
	height := len(pixels)
	width := len(pixels[0])
	if width/height != 2 {
		return fmt.Errorf("panorama must have 2:1 aspect ratio, got %d:%d", width, height)
	}
	return nil
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
	switch {
	case x < 1:
		x += float64(width)
	case x > float64(width):
		x -= float64(width)
	}
	return int(x)
}

func theta2Height(height int, theta float64) int {
	return int(float64(height) * theta / math.Pi)
}

func mapPixel(tileX, tileY int, pixels [][]Pixel, tile Tile, ac cache.AngleCache) Pixel {
	var theta, phi float64

	sphereHeight := len(pixels)
	sphereWidth := len(pixels[0])

	switch tile.Name {
	case FaceUp:
		theta = ac.ZenithPos[tileY][tileX]
		phi = ac.Azimuth[tileX][tileY]
		phi = updatePhi(tile.halfSize(), phi, tileY, tileX, math.Pi, 0, -math.Pi/2, math.Pi/2)
	case FaceDown:
		theta = ac.ZenithNeg[tileY][tileX]
		phi = ac.Azimuth[tileX][tile.Size-tileY-1]
		phi = updatePhi(tile.halfSize(), phi, tileY, tileX, 0, math.Pi, -math.Pi/2, math.Pi/2)
	case FaceFront:
		theta = ac.Polar[tile.Size-tileY-1][tile.Size-tileX-1]
		phi = ac.Azimuth[tileX][tile.Size-1]
		phi = updatePhi(tile.halfSize(), phi, tileY, tileX, 0, 0, -math.Pi/2, math.Pi/2)
	case FaceRight:
		theta = ac.Polar[tile.Size-tileY-1][tile.Size-tileX-1]
		phi = ac.Azimuth[tile.Size-1][tile.Size-tileX-1]
		phi = updatePhi(tile.halfSize(), phi, tileX, tileY, 0, math.Pi, math.Pi/2, math.Pi/2)
	case FaceBack:
		theta = ac.Polar[tile.Size-tileY-1][tile.Size-tileX-1]
		phi = ac.Azimuth[tileX][tile.Size-1] + math.Pi
	case FaceLeft:
		theta = ac.Polar[tile.Size-tileY-1][tile.Size-tileX-1]
		phi = ac.Azimuth[tile.Size-1][tile.Size-tileX-1]
		phi = updatePhi(tile.halfSize(), phi, tileX, tileY, math.Pi, 0, -math.Pi/2, -math.Pi/2)
	}

	spX := phi2Width(sphereWidth, phi)
	spY := theta2Height(sphereHeight, theta)

	return pixels[spY][spX]
}

// RenderTile generates a cubemap face image from the source panorama pixels.
func RenderTile(pixels [][]Pixel, tile Tile, ac cache.AngleCache) TileResult {
	tileImage := image.NewRGBA(image.Rect(0, 0, tile.Size, tile.Size))

	for tileY := range tile.Size {
		for tileX := range tile.Size {
			px := mapPixel(tileX, tileY, pixels, tile, ac)
			tileImage.Set(tileX, tileY, px.toColor())
		}
	}

	return TileResult{Tile: tile, Image: tileImage}
}

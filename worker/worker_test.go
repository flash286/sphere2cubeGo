package worker

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"sphere2cubeGo/cache"
)

func TestDecodePixels_JPEG(t *testing.T) {
	// Create a small 4x2 test image (2:1 ratio)
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(3, 1, color.RGBA{0, 0, 255, 255})

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}

	pixels, err := DecodePixels(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(pixels) != 2 {
		t.Errorf("expected 2 rows, got %d", len(pixels))
	}
	if len(pixels[0]) != 4 {
		t.Errorf("expected 4 cols, got %d", len(pixels[0]))
	}
}

func TestDecodePixels_Invalid(t *testing.T) {
	buf := bytes.NewBufferString("not an image")
	_, err := DecodePixels(buf)
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestValidateAspectRatio(t *testing.T) {
	tests := []struct {
		name    string
		w, h    int
		wantErr bool
	}{
		{"valid 4x2", 4, 2, false},
		{"valid 200x100", 200, 100, false},
		{"invalid 3x2", 3, 2, true},
		{"invalid 4x3", 4, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels := make([][]Pixel, tt.h)
			for y := range tt.h {
				pixels[y] = make([]Pixel, tt.w)
			}
			err := ValidateAspectRatio(pixels)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAspectRatio(%dx%d) error = %v, wantErr %v", tt.w, tt.h, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAspectRatio_Empty(t *testing.T) {
	err := ValidateAspectRatio(nil)
	if err == nil {
		t.Error("expected error for nil pixels")
	}
	err = ValidateAspectRatio([][]Pixel{})
	if err == nil {
		t.Error("expected error for empty pixels")
	}
}

func TestRenderTile_OutputDimensions(t *testing.T) {
	// Create a minimal 4x2 panorama
	pixels := make([][]Pixel, 2)
	for y := range 2 {
		pixels[y] = make([]Pixel, 4)
		for x := range 4 {
			pixels[y][x] = Pixel{128, 128, 128, 255}
		}
	}

	tileSize := 2
	ac := cache.Precompute(tileSize)

	for _, face := range AllFaces() {
		tile := Tile{Size: tileSize, Name: face}
		result := RenderTile(pixels, tile, ac)

		bounds := result.Image.Bounds()
		if bounds.Dx() != tileSize || bounds.Dy() != tileSize {
			t.Errorf("face %s: expected %dx%d, got %dx%d",
				face, tileSize, tileSize, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestAllFaces(t *testing.T) {
	faces := AllFaces()
	if len(faces) != 6 {
		t.Errorf("expected 6 faces, got %d", len(faces))
	}
	expected := map[string]bool{
		FaceUp: true, FaceDown: true, FaceFront: true,
		FaceRight: true, FaceBack: true, FaceLeft: true,
	}
	for _, f := range faces {
		if !expected[f] {
			t.Errorf("unexpected face: %s", f)
		}
	}
}

func TestPixelToColor(t *testing.T) {
	p := Pixel{255, 128, 0, 255}
	c := p.toColor()
	r, g, b, a := c.RGBA()
	// Values are scaled by 257
	if r != 65535 || b != 0 || a != 65535 {
		t.Errorf("unexpected RGBA: %d, %d, %d, %d", r, g, b, a)
	}
	_ = g // 128*257 = 32896
}

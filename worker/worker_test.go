package worker

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"sphere2cubeGo/cache"
)

func makeTestPanorama(width, height int) *Panorama {
	pix := make([]uint8, width*height*4)
	for y := range height {
		for x := range width {
			off := (y*width + x) * 4
			pix[off] = uint8(255 * x / width)
			pix[off+1] = uint8(255 * y / height)
			pix[off+2] = 128
			pix[off+3] = 255
		}
	}
	return NewPanorama(width, height, pix)
}

func TestDecodeImage_JPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(3, 1, color.RGBA{0, 0, 255, 255})

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}

	pano, err := DecodeImage(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if pano.Width != 4 || pano.Height != 2 {
		t.Errorf("expected 4x2, got %dx%d", pano.Width, pano.Height)
	}
	if len(pano.Pix) != 4*2*4 {
		t.Errorf("expected %d bytes, got %d", 4*2*4, len(pano.Pix))
	}
}

func TestDecodeImage_Invalid(t *testing.T) {
	buf := bytes.NewBufferString("not an image")
	_, err := DecodeImage(buf)
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
			pano := makeTestPanorama(tt.w, tt.h)
			err := pano.ValidateAspectRatio()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAspectRatio(%dx%d) error = %v, wantErr %v", tt.w, tt.h, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAspectRatio_Empty(t *testing.T) {
	pano := &Panorama{}
	if err := pano.ValidateAspectRatio(); err == nil {
		t.Error("expected error for empty panorama")
	}
}

func TestRenderTile_OutputDimensions(t *testing.T) {
	pano := makeTestPanorama(4, 2)
	tileSize := 2
	pm := cache.Precompute(tileSize, pano.Width, pano.Height)

	for _, face := range AllFaces() {
		tile := Tile{Size: tileSize, Name: face}
		result := RenderTile(pano, tile, &pm)

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

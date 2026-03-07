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

// makeCoordPanorama creates a panorama where pixel color encodes its position:
// R = x * 255 / width, G = y * 255 / height, B = 128, A = 255.
// This lets us verify that rendered tile pixels map to the correct panorama location.
func makeCoordPanorama(width, height int) *Panorama {
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

func TestRenderTile_CenterPixelMapping(t *testing.T) {
	// Use a large enough panorama and tile for meaningful center pixel checks.
	panoW, panoH := 1024, 512
	tileSize := 128
	pano := makeCoordPanorama(panoW, panoH)
	pm := cache.Precompute(tileSize, panoW, panoH)

	center := tileSize / 2

	// For a coordinate panorama:
	//   R = panoX * 255 / panoW  → panoX ≈ R * panoW / 255
	//   G = panoY * 255 / panoH  → panoY ≈ G * panoH / 255
	//
	// Expected center pixel panorama coordinates:
	//   front:  (panoW/2, panoH/2) — phi=0, theta=pi/2 (equator center)
	//   back:   (0 or panoW, panoH/2) — phi=pi (equator opposite side)
	//   right:  (3*panoW/4, panoH/2) — phi=pi/2 (equator 90° right)
	//   left:   (panoW/4, panoH/2) — phi=-pi/2 (equator 90° left)
	//   up:     (*, 0) — theta≈0 (north pole, any longitude)
	//   down:   (*, panoH) — theta≈pi (south pole, any longitude)

	tests := []struct {
		face   string
		checkR bool // whether to check R (longitude)
		wantR  uint8
		checkG bool // whether to check G (latitude)
		wantG  uint8
		tolR   uint8
		tolG   uint8
	}{
		// Front face center → equator center of panorama
		{FaceFront, true, 127, true, 127, 10, 10},
		// Back face center → equator, opposite side (x≈0 or x≈panoW, wraps)
		{FaceBack, false, 0, true, 127, 0, 10},
		// Right face center → equator, 3/4 along width
		{FaceRight, true, 191, true, 127, 10, 10},
		// Left face center → equator, 1/4 along width
		{FaceLeft, true, 63, true, 127, 10, 10},
		// Up face center → near north pole (G ≈ 0)
		{FaceUp, false, 0, true, 0, 0, 15},
		// Down face center → near south pole (G ≈ 255)
		{FaceDown, false, 0, true, 255, 0, 15},
	}

	for _, tt := range tests {
		tile := Tile{Size: tileSize, Name: tt.face}
		result := RenderTile(pano, tile, &pm)

		off := (center*tileSize + center) * 4
		gotR := result.Image.Pix[off]
		gotG := result.Image.Pix[off+1]
		gotA := result.Image.Pix[off+3]

		if gotA != 255 {
			t.Errorf("%s center: alpha=%d, want 255", tt.face, gotA)
		}
		if tt.checkR {
			diff := int(gotR) - int(tt.wantR)
			if diff < 0 {
				diff = -diff
			}
			if diff > int(tt.tolR) {
				t.Errorf("%s center: R=%d, want %d±%d (pano X mapping)",
					tt.face, gotR, tt.wantR, tt.tolR)
			}
		}
		if tt.checkG {
			diff := int(gotG) - int(tt.wantG)
			if diff < 0 {
				diff = -diff
			}
			if diff > int(tt.tolG) {
				t.Errorf("%s center: G=%d, want %d±%d (pano Y mapping)",
					tt.face, gotG, tt.wantG, tt.tolG)
			}
		}
	}
}

func TestRenderTile_AllPixelsOpaque(t *testing.T) {
	pano := makeCoordPanorama(512, 256)
	tileSize := 64
	pm := cache.Precompute(tileSize, pano.Width, pano.Height)

	for _, face := range AllFaces() {
		tile := Tile{Size: tileSize, Name: face}
		result := RenderTile(pano, tile, &pm)

		for i := 3; i < len(result.Image.Pix); i += 4 {
			if result.Image.Pix[i] != 255 {
				px := (i / 4) % tileSize
				py := (i / 4) / tileSize
				t.Errorf("%s pixel (%d,%d): alpha=%d, want 255",
					face, px, py, result.Image.Pix[i])
				break
			}
		}
	}
}

func TestRenderTile_FacesAreDifferent(t *testing.T) {
	pano := makeCoordPanorama(512, 256)
	tileSize := 32
	pm := cache.Precompute(tileSize, pano.Width, pano.Height)

	faces := AllFaces()
	images := make(map[string][]uint8)
	for _, face := range faces {
		tile := Tile{Size: tileSize, Name: face}
		result := RenderTile(pano, tile, &pm)
		dst := make([]uint8, len(result.Image.Pix))
		copy(dst, result.Image.Pix)
		images[face] = dst
	}

	// Each face should differ from every other face
	for i := 0; i < len(faces); i++ {
		for j := i + 1; j < len(faces); j++ {
			a, b := images[faces[i]], images[faces[j]]
			same := 0
			for k := range a {
				if a[k] == b[k] {
					same++
				}
			}
			// Allow some coincidental matches but faces should be mostly different
			ratio := float64(same) / float64(len(a))
			if ratio > 0.95 {
				t.Errorf("%s and %s are %.0f%% identical — likely a mapping bug",
					faces[i], faces[j], ratio*100)
			}
		}
	}
}

func TestRenderTile_Deterministic(t *testing.T) {
	pano := makeCoordPanorama(256, 128)
	tileSize := 32
	pm := cache.Precompute(tileSize, pano.Width, pano.Height)

	tile := Tile{Size: tileSize, Name: FaceFront}
	r1 := RenderTile(pano, tile, &pm)
	r2 := RenderTile(pano, tile, &pm)

	for i := range r1.Image.Pix {
		if r1.Image.Pix[i] != r2.Image.Pix[i] {
			t.Fatal("RenderTile is not deterministic")
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

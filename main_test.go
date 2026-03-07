package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"sphere2cubeGo/cache"
	"sphere2cubeGo/saver"
	"sphere2cubeGo/worker"
)

// createTestPanorama generates a synthetic equirectangular panorama (2:1 ratio)
// with a color gradient so each cubemap face produces visually distinct output.
func createTestPanorama(t *testing.T, width, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			r := uint8(255 * x / width)
			g := uint8(255 * y / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	path := filepath.Join(t.TempDir(), "panorama.jpg")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEndToEnd(t *testing.T) {
	// 1. Create a synthetic 200x100 panorama (2:1)
	panoramaPath := createTestPanorama(t, 200, 100)
	outputDir := filepath.Join(t.TempDir(), "output")
	tileSize := 64
	quality := 90

	// 2. Decode input
	f, err := os.Open(panoramaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pixels, err := worker.DecodePixels(f)
	if err != nil {
		t.Fatalf("DecodePixels: %v", err)
	}

	if err := worker.ValidateAspectRatio(pixels); err != nil {
		t.Fatalf("ValidateAspectRatio: %v", err)
	}

	// 3. Pre-compute cache and render all 6 faces
	ac := cache.Precompute(tileSize)
	faces := worker.AllFaces()

	for _, name := range faces {
		tile := worker.Tile{Size: tileSize, Name: name}
		result := worker.RenderTile(pixels, tile, ac)

		// 4. Save each tile
		if err := saver.SaveTile(result, outputDir, quality); err != nil {
			t.Fatalf("SaveTile(%s): %v", name, err)
		}
	}

	// 5. Verify all 6 output files exist and are valid JPEGs
	for _, name := range faces {
		path := filepath.Join(outputDir, name+".jpg")

		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("face %s: file not found: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("face %s: file is empty", name)
			continue
		}

		out, err := os.Open(path)
		if err != nil {
			t.Errorf("face %s: %v", name, err)
			continue
		}

		img, err := jpeg.Decode(out)
		out.Close()
		if err != nil {
			t.Errorf("face %s: invalid JPEG: %v", name, err)
			continue
		}

		bounds := img.Bounds()
		if bounds.Dx() != tileSize || bounds.Dy() != tileSize {
			t.Errorf("face %s: expected %dx%d, got %dx%d",
				name, tileSize, tileSize, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestEndToEnd_LargerTile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping larger tile test in short mode")
	}

	panoramaPath := createTestPanorama(t, 1024, 512)
	outputDir := filepath.Join(t.TempDir(), "output")
	tileSize := 256

	f, err := os.Open(panoramaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pixels, err := worker.DecodePixels(f)
	if err != nil {
		t.Fatal(err)
	}

	ac := cache.Precompute(tileSize)

	for _, name := range worker.AllFaces() {
		tile := worker.Tile{Size: tileSize, Name: name}
		result := worker.RenderTile(pixels, tile, ac)

		if err := saver.SaveTile(result, outputDir, 100); err != nil {
			t.Fatalf("SaveTile(%s): %v", name, err)
		}
	}

	// Verify all files exist
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Errorf("expected 6 files, got %d", len(entries))
	}
}

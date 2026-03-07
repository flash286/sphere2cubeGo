package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
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
	panoramaPath := createTestPanorama(t, 200, 100)
	outputDir := filepath.Join(t.TempDir(), "output")
	tileSize := 64
	quality := 90

	f, err := os.Open(panoramaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pano, err := worker.DecodeImage(f)
	if err != nil {
		t.Fatalf("DecodeImage: %v", err)
	}

	if err := pano.ValidateAspectRatio(); err != nil {
		t.Fatalf("ValidateAspectRatio: %v", err)
	}

	ac := cache.Precompute(tileSize)
	faces := worker.AllFaces()

	for _, name := range faces {
		tile := worker.Tile{Size: tileSize, Name: name}
		result := worker.RenderTile(pano, tile, ac)

		if err := saver.SaveTile(result, outputDir, quality); err != nil {
			t.Fatalf("SaveTile(%s): %v", name, err)
		}
	}

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

	pano, err := worker.DecodeImage(f)
	if err != nil {
		t.Fatal(err)
	}

	ac := cache.Precompute(tileSize)

	for _, name := range worker.AllFaces() {
		tile := worker.Tile{Size: tileSize, Name: name}
		result := worker.RenderTile(pano, tile, ac)

		if err := saver.SaveTile(result, outputDir, 100); err != nil {
			t.Fatalf("SaveTile(%s): %v", name, err)
		}
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Errorf("expected 6 files, got %d", len(entries))
	}
}

// createGradientPanorama builds an in-memory 2:1 panorama directly,
// bypassing JPEG encode/decode to keep benchmark setup overhead minimal.
func createGradientPanorama(width, height int) *worker.Panorama {
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
	return worker.NewPanorama(width, height, pix)
}

// BenchmarkPrecompute measures angle cache generation time.
func BenchmarkPrecompute(b *testing.B) {
	for _, tileSize := range []int{256, 512, 1024} {
		b.Run(sizeLabel(tileSize), func(b *testing.B) {
			for range b.N {
				cache.Precompute(tileSize)
			}
		})
	}
}

// BenchmarkRenderTile measures single-tile rendering (front face).
func BenchmarkRenderTile(b *testing.B) {
	for _, tc := range []struct{ panoW, panoH, tile int }{
		{2048, 1024, 512},
		{4096, 2048, 1024},
	} {
		pano := createGradientPanorama(tc.panoW, tc.panoH)
		ac := cache.Precompute(tc.tile)
		tile := worker.Tile{Size: tc.tile, Name: worker.FaceFront}

		b.Run(sizeLabel(tc.tile)+"_pano"+sizeLabel(tc.panoW), func(b *testing.B) {
			for range b.N {
				worker.RenderTile(pano, tile, ac)
			}
		})
	}
}

// BenchmarkFullPipeline measures the full pipeline: cache + render all 6 faces + save.
func BenchmarkFullPipeline(b *testing.B) {
	for _, tc := range []struct{ panoW, panoH, tile int }{
		{2048, 1024, 256},
		{4096, 2048, 512},
		{4096, 2048, 1024},
	} {
		pano := createGradientPanorama(tc.panoW, tc.panoH)

		label := "pano" + sizeLabel(tc.panoW) + "_tile" + sizeLabel(tc.tile)
		b.Run(label, func(b *testing.B) {
			for range b.N {
				outputDir := b.TempDir()
				ac := cache.Precompute(tc.tile)
				faces := worker.AllFaces()
				results := make([]worker.TileResult, len(faces))

				var wg sync.WaitGroup
				for i, name := range faces {
					wg.Add(1)
					go func(idx int, faceName string) {
						defer wg.Done()
						t := worker.Tile{Size: tc.tile, Name: faceName}
						results[idx] = worker.RenderTile(pano, t, ac)
					}(i, name)
				}
				wg.Wait()

				for _, r := range results {
					if err := saver.SaveTile(r, outputDir, 90); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func sizeLabel(n int) string {
	if n >= 1024 && n%1024 == 0 {
		return itoa(n/1024) + "k"
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

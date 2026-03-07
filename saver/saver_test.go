package saver

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"sphere2cubeGo/worker"
)

func TestSaveTile(t *testing.T) {
	dir := t.TempDir()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})

	result := worker.TileResult{
		Tile:  worker.Tile{Size: 4, Name: "front"},
		Image: img,
	}

	if err := SaveTile(result, dir, 90); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "front.jpg")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestSaveTile_CreatesSubdirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "output")

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	result := worker.TileResult{
		Tile:  worker.Tile{Size: 2, Name: "up"},
		Image: img,
	}

	if err := SaveTile(result, dir, 100); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "up.jpg")); err != nil {
		t.Fatalf("file not created in nested dir: %v", err)
	}
}

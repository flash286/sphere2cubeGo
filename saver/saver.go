package saver

import (
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	"sphere2cubeGo/worker"
)

// SaveTile writes a tile result to a JPEG file in the given directory.
func SaveTile(result worker.TileResult, outputDir string, quality int) error {
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	path := filepath.Join(outputDir, result.Tile.Name+".jpg")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, result.Image, &jpeg.Options{Quality: quality}); err != nil {
		return fmt.Errorf("encode JPEG: %w", err)
	}

	return nil
}

package saver

import (
	"os"
	"image/jpeg"
	"path/filepath"
	"sphere2cubeGo/worker"
)

func SaveTile(tileResult worker.TileResult, outPutDir string) error {

	err := os.Mkdir(outPutDir, os.FileMode(os.ModePerm))

	if err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}

	finalPath := filepath.Join(outPutDir, tileResult.Tile.TileName+".jpg")

	f, err := os.Create(finalPath)

	if err != nil {
		return err
	}

	defer f.Close()
	jpeg.Encode(f, tileResult.Image, &jpeg.Options{100})

	return nil
}


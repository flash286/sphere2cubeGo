package main

import (
	"log"
	"os"
	"sphere2cubeGo/cache"
	"time"
	"flag"
	"sphere2cubeGo/worker"
	"sphere2cubeGo/saver"
)

var (
	tileNames = []string{
		worker.TileUp,
		worker.TileDown,
		worker.TileFront,
		worker.TileRight,
		worker.TileBack,
		worker.TileLeft,
	}
	tileSize          = 1024
	originalImagePath = ""
	outPutDir         = "result"
)

func main() {

	tileSizeCmd := flag.Int("tile_size", tileSize, "Size in px of final tile")
	originalImagePathCmd := flag.String("input_path", "", "")
	outPutDirCmd := flag.String("output_path", outPutDir, "")

	flag.Parse()

	tileSize = *tileSizeCmd
	originalImagePath = *originalImagePathCmd
	outPutDir = *outPutDirCmd

	if originalImagePath == "" {
		log.Fatal("input_path is required")
		os.Exit(1)
	}

	_, err := os.Stat(originalImagePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("%v not found", originalImagePath)
		}
	}

	done := make(chan worker.TileResult)
	timeStart := time.Now()
	cacheResult := cache.CacheAnglesHandler(tileSize)
	for _, tileName := range tileNames {
		tile := worker.Tile{TileName: tileName, TileSize: tileSize}
		go worker.Worker(tile, cacheResult, originalImagePath, done)
	}

	for range tileNames {
		tileResult := <-done
		err = saver.SaveTile(tileResult, outPutDir)

		if err != nil {
			panic(err)
		}

		log.Printf("Process for tile %v --> finished", tileResult.Tile.TileName)
	}

	timeFinish := time.Now()
	duration := timeFinish.Sub(timeStart)
	log.Printf("Time to render: %v seconds", duration.Seconds())
}

package main

import (
	"flag"
	"log"
	"os"
	"sphere2cubeGo/cache"
	"sphere2cubeGo/saver"
	"sphere2cubeGo/worker"
	"time"
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
	outPutDir         = "./build"

	tileSizeCmd          = flag.Int("s", tileSize, "Size in px of final tile")
	originalImagePathCmd = flag.String("i", "", "Path to input equirectangular panorama")
	outPutDirCmd         = flag.String("o", outPutDir, "Path to output directory")
)

func main() {

	flag.Parse()
	tileSize = *tileSizeCmd
	originalImagePath = *originalImagePathCmd
	outPutDir = *outPutDirCmd

	if originalImagePath == "" {
		flag.PrintDefaults()
		os.Exit(2)
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
			log.Fatal(err.Error())
			os.Exit(2)
		}

		log.Printf("Process for tile %v --> finished", tileResult.Tile.TileName)
	}

	timeFinish := time.Now()
	duration := timeFinish.Sub(timeStart)
	log.Printf("Time to render: %v seconds", duration.Seconds())
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"sphere2cubeGo/cache"
	"sphere2cubeGo/saver"
	"sphere2cubeGo/worker"
)

func run() error {
	tileSize := flag.Int("s", 1024, "Size in px of final tile")
	inputPath := flag.String("i", "", "Path to input equirectangular panorama (JPEG or PNG)")
	outputDir := flag.String("o", "./build", "Path to output directory")
	quality := flag.Int("q", 100, "JPEG output quality (1-100)")
	flag.Parse()

	if *inputPath == "" {
		flag.PrintDefaults()
		return fmt.Errorf("input path is required (-i)")
	}

	if *quality < 1 || *quality > 100 {
		return fmt.Errorf("quality must be between 1 and 100, got %d", *quality)
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer func() { _ = f.Close() }()

	start := time.Now()

	log.Printf("Decoding %s", *inputPath)
	pano, err := worker.DecodeImage(f)
	if err != nil {
		return fmt.Errorf("decode input: %w", err)
	}

	if err := pano.ValidateAspectRatio(); err != nil {
		return err
	}

	log.Printf("Pre-computing angle cache for tile size %d", *tileSize)
	angleCache := cache.Precompute(*tileSize)

	faces := worker.AllFaces()
	results := make([]worker.TileResult, len(faces))

	var wg sync.WaitGroup
	for i, name := range faces {
		wg.Add(1)
		go func(idx int, faceName string) {
			defer wg.Done()
			tile := worker.Tile{Size: *tileSize, Name: faceName}
			log.Printf("Rendering tile %s", faceName)
			results[idx] = worker.RenderTile(pano, tile, &angleCache)
			log.Printf("Tile %s done", faceName)
		}(i, name)
	}
	wg.Wait()

	saveErrs := make([]error, len(results))
	var saveWg sync.WaitGroup
	for i, result := range results {
		saveWg.Add(1)
		go func(idx int, r worker.TileResult) {
			defer saveWg.Done()
			saveErrs[idx] = saver.SaveTile(r, *outputDir, *quality)
		}(i, result)
	}
	saveWg.Wait()
	for i, err := range saveErrs {
		if err != nil {
			return fmt.Errorf("save tile %s: %w", results[i].Tile.Name, err)
		}
	}

	log.Printf("Completed in %.2f seconds", time.Since(start).Seconds())
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

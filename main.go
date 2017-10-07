package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"math"
	"os"
	"sphere2cube/cache"
	"time"
)

const (
	TileUp    = "up"
	TileDown  = "down"
	TileFront = "front"
	TileRight = "right"
	TileBack  = "back"
	TileLeft  = "left"
)

var (
	tileNames         = []string{TileUp, TileDown, TileFront, TileRight, TileBack, TileLeft}
	tileSize          = 512
	originalImagePath = "/panorama.jpg"
)

// Pixel struct
type Pixel struct {
	R int
	G int
	B int
	A int
}

func (pixel *Pixel) pixelToRGBA() color.Color {
	return color.RGBA64{uint16(pixel.R * 257), uint16(pixel.G * 257), uint16(pixel.B * 257), uint16(pixel.A * 257)}
}

func getHalfSize() float64 {
	return float64(tileSize-1) / 2
}

// img.At(x, y).RGBA() returns four uint32 values; we want a Pixel
func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel {
	return Pixel{int(r / 257), int(g / 257), int(b / 257), int(a / 257)}
}

// Get the bi-dimensional pixel array
func getPixels(file io.Reader) ([][]Pixel, error) {
	img, err := jpeg.Decode(file)

	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]Pixel
	for y := 0; y < height; y++ {
		var row []Pixel
		for x := 0; x < width; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return pixels, nil
}

func updatePhi(half_size float64, phi float64, major_dir int, minor_dir int, major_m float64, major_p float64, minor_m float64, minor_p float64) float64 {

	if float64(major_dir) < half_size {
		phi = phi + major_m
	} else if float64(major_dir) > half_size {
		phi = phi + major_p
	} else if float64(minor_dir) < half_size {
		phi = minor_m
	} else {
		phi = minor_p
	}

	return phi
}

func phi2Width(width int, phi float64) uint8 {
	x := 0.5 * float64(width) * (phi/math.Pi + 1)

	if x < 1 {
		x += float64(width)
	} else if x > float64(width) {
		x -= float64(width)
	}

	return uint8(x)
}

func theta2Height(height int, theta float64) uint8 {
	return uint8(float64(height) * theta / math.Pi)
}

func processCords(tileX int, tileY int, originalImage [][]Pixel, tileName string, mathCache cache.CacheAngles) Pixel {

	theta := 0.0
	phi := 0.0

	sphereHeight, sphereWidth := len(originalImage), len(originalImage[0])

	if tileName == TileUp {
		theta = mathCache.ZP[tileY][tileX]
		phi = mathCache.PHI[tileX][tileY]
		phi = updatePhi(getHalfSize(), phi, tileY, tileX, math.Pi, 0, -math.Pi/2, math.Pi/2)
	} else if tileName == TileDown {
		theta = mathCache.ZM[tileY][tileX]
		phi = mathCache.PHI[tileX][tileSize-tileY-1]
		phi = updatePhi(getHalfSize(), phi, tileY, tileX, 0, math.Pi, -math.Pi/2, math.Pi/2)
	} else if tileName == TileFront {
		theta = mathCache.XYPM[tileSize-tileY-1][tileSize-tileX-1]
		phi = mathCache.PHI[tileX][tileSize-1] //tile_x, tile_size - 1
		phi = updatePhi(getHalfSize(), phi, tileY, tileX, 0, 0, -math.Pi/2, math.Pi/2)
	} else if tileName == TileRight {
		theta = mathCache.XYPM[tileSize-tileY-1][tileSize-tileX-1]
		phi = mathCache.PHI[tileSize-1][tileSize-tileX-1]
		phi = updatePhi(getHalfSize(), phi, tileX, tileY, 0, math.Pi, math.Pi/2, math.Pi/2)
	} else if tileName == TileBack {
		theta = mathCache.XYPM[tileSize-tileY-1][tileSize-tileX-1]
		phi = mathCache.PHI[tileX][tileSize-1] + math.Pi
	} else if tileName == TileLeft {
		theta = mathCache.XYPM[tileSize-tileY-1][tileSize-tileX-1]
		phi = mathCache.PHI[tileSize-1][tileSize-tileX-1]
		phi = updatePhi(getHalfSize(), phi, tileX, tileY, math.Pi, 0, -math.Pi/2, -math.Pi/2)
	}

	spX, spY := phi2Width(sphereWidth, phi), theta2Height(sphereHeight, theta)

	//log.Printf("[%v]Theta: %v", tileName, theta)
	//log.Printf("[%v]phi: %v", tileName, phi)

	//theta + phi

	return originalImage[spY][spX]
}

func worker(tileName string, mathCache cache.CacheAngles, tileSize int, originalImagePath string, done chan string) {
	log.Printf("Process for tile %v --> started", tileName)

	tile := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))

	pwd, _ := os.Getwd()

	reader, err := os.Open(pwd + originalImagePath)

	if err != nil {
		panic(err)
	}

	defer reader.Close()

	originalPixels, err := getPixels(reader)

	if err != nil {
		panic(err)
	}

	for tileY := 0; tileY < tileSize; tileY++ {
		for tileX := 0; tileX < tileSize; tileX++ {
			pixelToMove := processCords(tileX, tileY, originalPixels, tileName, mathCache)
			colorPixel := pixelToMove.pixelToRGBA()
			tile.Set(tileY, tileX, colorPixel)
		}
	}

	f, err := os.Create(tileName + ".jpg")

	if err != nil {
		panic(err)
	}

	defer f.Close()
	jpeg.Encode(f, tile, &jpeg.Options{100})

	done <- tileName

}

func main() {
	done := make(chan string)

	timeStart := time.Now()

	cacheResult := cache.CacheAnglesHandler(tileSize)
	for _, tileName := range tileNames {
		go worker(tileName, cacheResult, tileSize, originalImagePath, done)
		value := <-done
		log.Printf("Process for tile %v --> finished", value)
	}

	for range tileNames {

	}

	timeFinish := time.Now()

	duration := timeFinish.Sub(timeStart)

	log.Printf("Time to render: %v seconds", duration.Seconds())
}

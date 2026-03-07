<p align="center">
  <img src="pic-to-explain.png" alt="Equirectangular to Cubemap" width="600">
</p>

<h1 align="center">sphere2cubeGo</h1>

<p align="center">
  Fast, zero-dependency CLI tool to convert equirectangular panoramas into cubemap faces.
</p>

<p align="center">
  <a href="https://github.com/flash286/sphere2cubeGo/actions/workflows/ci.yml"><img src="https://github.com/flash286/sphere2cubeGo/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/flash286/sphere2cubeGo"><img src="https://goreportcard.com/badge/github.com/flash286/sphere2cubeGo" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go 1.22+">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"></a>
</p>

---

## What it does

Takes a single equirectangular (2:1 ratio) panoramic image and produces six cubemap face images — ready for OpenGL, game engines, or any 3D renderer that uses cube-mapped textures.

**Input:** one panorama (JPEG or PNG) &rarr; **Output:** `up.jpg`, `down.jpg`, `front.jpg`, `back.jpg`, `left.jpg`, `right.jpg`

## Features

- **Zero external dependencies** — only Go standard library
- **Parallel rendering** — all 6 faces rendered concurrently via goroutines
- **Pre-computed trig cache** — angle lookup tables eliminate redundant `math.Acos`/`math.Atan` calls
- **Direct pixel manipulation** — writes to raw `Pix` byte slices, no per-pixel allocations
- **Cross-platform** — builds for Linux, macOS, and Windows (amd64, arm64)

## Performance

Benchmarked on Intel Xeon Platinum 8581C:

| Panorama | Tile size | 6 faces rendered in |
|----------|-----------|---------------------|
| 2048x1024 | 256px | ~27ms |
| 4096x2048 | 512px | ~100ms |
| 4096x2048 | 1024px | ~337ms |

## Installation

```bash
go install github.com/flash286/sphere2cubeGo@latest
```

Or build from source:

```bash
git clone https://github.com/flash286/sphere2cubeGo.git
cd sphere2cubeGo
go build -o sphere2cubeGo .
```

## Usage

Basic — converts panorama to 1024px cubemap faces in `./build/`:

```bash
./sphere2cubeGo -i panorama.jpg
```

Custom tile size:

```bash
./sphere2cubeGo -i panorama.jpg -s 2048
```

Custom output directory and JPEG quality:

```bash
./sphere2cubeGo -i panorama.jpg -s 2048 -o /path/to/output -q 95
```

### All flags

| Flag | Default | Description |
|------|---------|-------------|
| `-i` | *(required)* | Path to input equirectangular panorama (JPEG or PNG) |
| `-s` | `1024` | Size in pixels of each output face |
| `-o` | `./build` | Output directory |
| `-q` | `100` | JPEG output quality (1–100) |

## How it works

1. **Decode** the equirectangular panorama into a flat pixel buffer
2. **Pre-compute** trigonometric lookup tables (acos, atan) for the given tile size
3. **Map** each pixel of each cube face back to spherical coordinates, then to the source panorama pixel
4. **Write** the six output JPEG files

### The math

The core problem: for each pixel on a cube face, find which pixel in the source panorama it corresponds to. This is a two-step coordinate transformation.

#### Step 1 — Cube face pixel to spherical coordinates

Each cube face pixel `(tx, ty)` is normalized to the `[-1, 1]` range:

```
x = tx / halfSize - 1
y = ty / halfSize - 1
```

These represent 3D direction vectors from the cube center through the face. The specific mapping depends on which face we're rendering — for example, the **front** face has an implicit `z = 1`, the **right** face has `x = 1`, etc.

From the direction vector we compute the distance to the origin and the two spherical angles:

```
r = sqrt(x² + y² + 1)

theta (polar angle, 0 at top pole to π at bottom):
  - Up face:   theta = acos(1 / r)     — looking up, z = +1
  - Down face: theta = acos(-1 / r)    — looking down, z = -1
  - Side faces: theta = acos(y / r)    — y maps to vertical

phi (azimuth, -π to π around the equator):
  - Base:  phi = atan(y / x)
  - Adjusted per face by adding offsets (0, ±π/2, π)
    to rotate into the correct quadrant
```

The `atan + offset` logic (see `updatePhi` in code) resolves the quadrant ambiguity that `atan` alone cannot handle — it checks whether the pixel is above/below or left/right of the face center and applies the correct correction.

#### Step 2 — Spherical coordinates to panorama pixel

The equirectangular projection maps longitude linearly to x and latitude linearly to y:

```
x_pano = width × (phi / π + 1) / 2
y_pano = height × theta / π
```

This is the [equirectangular projection](https://en.wikipedia.org/wiki/Equirectangular_projection) inverse — `phi ∈ [-π, π]` maps to `[0, width]` and `theta ∈ [0, π]` maps to `[0, height]`.

Values that fall outside `[1, width]` are wrapped around (the panorama repeats horizontally).

#### Pre-computed cache

Since `r`, `theta`, and `phi` only depend on the tile pixel position (not on the panorama content), all trig values are computed once in `cache.Precompute()` and stored in flat lookup tables:

| Table | Formula | Used by |
|-------|---------|---------|
| `ZenithPos` | `acos(1/r)` | Up face |
| `ZenithNeg` | `acos(-1/r)` | Down face |
| `Polar` | `acos(y/r)` | Front, Right, Back, Left |
| `Azimuth` | `atan(y/x)` | All faces |

This trades ~32MB of memory (for 1024px tiles) for eliminating 4 trig calls per pixel × 6M pixels = 24M trig operations.

## Project structure

```
.
├── main.go              # CLI entry point, flag parsing, orchestration
├── worker/
│   └── worker.go        # Core rendering: panorama decoding, pixel mapping, tile generation
├── cache/
│   └── cache.go         # Pre-computed trigonometric lookup tables
├── saver/
│   └── saver.go         # JPEG output writer
├── main_test.go         # E2E and benchmark tests
├── *_test.go            # Unit tests in each package
└── .github/workflows/
    └── ci.yml           # CI: test, vet, cross-platform build
```

## Running tests

```bash
# Unit + E2E tests
go test ./...

# Benchmarks
go test -bench=. -benchmem -run=^$

# Skip slow tests
go test ./... -short
```

## License

[Apache License 2.0](LICENSE)

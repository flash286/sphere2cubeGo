package cache

import (
	"math"
	"testing"
)

func TestPrecompute_Dimensions(t *testing.T) {
	sizes := []int{4, 16, 64}
	for _, size := range sizes {
		c := Precompute(size)
		for _, table := range [][][]float64{c.ZenithPos, c.ZenithNeg, c.Polar, c.Azimuth} {
			if len(table) != size {
				t.Errorf("expected %d rows, got %d", size, len(table))
			}
			for y, row := range table {
				if len(row) != size {
					t.Errorf("row %d: expected %d cols, got %d", y, size, len(row))
				}
			}
		}
	}
}

func TestPrecompute_CenterValues(t *testing.T) {
	size := 11 // odd so center is exact
	c := Precompute(size)
	center := size / 2

	// At center (x=0, y=0), root=1, so ZenithPos = acos(1) = 0
	if got := c.ZenithPos[center][center]; math.Abs(got) > 1e-10 {
		t.Errorf("ZenithPos at center: expected ~0, got %f", got)
	}

	// ZenithNeg at center = acos(-1) = Pi
	if got := c.ZenithNeg[center][center]; math.Abs(got-math.Pi) > 1e-10 {
		t.Errorf("ZenithNeg at center: expected ~Pi, got %f", got)
	}

	// Azimuth at center (x=0) should be 0 (special case)
	if got := c.Azimuth[center][center]; got != 0 {
		t.Errorf("Azimuth at center: expected 0, got %f", got)
	}
}

func TestPrecompute_ValuesInRange(t *testing.T) {
	c := Precompute(32)
	for y := range 32 {
		for x := range 32 {
			if v := c.ZenithPos[y][x]; v < 0 || v > math.Pi {
				t.Errorf("ZenithPos[%d][%d] = %f out of [0, Pi]", y, x, v)
			}
			if v := c.ZenithNeg[y][x]; v < 0 || v > math.Pi {
				t.Errorf("ZenithNeg[%d][%d] = %f out of [0, Pi]", y, x, v)
			}
			if v := c.Polar[y][x]; v < 0 || v > math.Pi {
				t.Errorf("Polar[%d][%d] = %f out of [0, Pi]", y, x, v)
			}
		}
	}
}

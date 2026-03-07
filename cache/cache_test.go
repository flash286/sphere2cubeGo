package cache

import (
	"math"
	"testing"
)

func TestPrecompute_Dimensions(t *testing.T) {
	sizes := []int{4, 16, 64}
	for _, size := range sizes {
		c := Precompute(size)
		n := size * size
		for _, table := range [][]float64{c.ZenithPos, c.ZenithNeg, c.Polar, c.Azimuth} {
			if len(table) != n {
				t.Errorf("size=%d: expected %d elements, got %d", size, n, len(table))
			}
		}
	}
}

func TestPrecompute_CenterValues(t *testing.T) {
	size := 11 // odd so center is exact
	c := Precompute(size)
	center := c.At(size/2, size/2)

	// At center (x=0, y=0), root=1, so ZenithPos = acos(1) = 0
	if got := c.ZenithPos[center]; math.Abs(got) > 1e-10 {
		t.Errorf("ZenithPos at center: expected ~0, got %f", got)
	}

	// ZenithNeg at center = acos(-1) = Pi
	if got := c.ZenithNeg[center]; math.Abs(got-math.Pi) > 1e-10 {
		t.Errorf("ZenithNeg at center: expected ~Pi, got %f", got)
	}

	// Azimuth at center (x=0) should be 0 (special case)
	if got := c.Azimuth[center]; got != 0 {
		t.Errorf("Azimuth at center: expected 0, got %f", got)
	}
}

func TestPrecompute_ValuesInRange(t *testing.T) {
	size := 32
	c := Precompute(size)
	for i := range size * size {
		if v := c.ZenithPos[i]; v < 0 || v > math.Pi {
			t.Errorf("ZenithPos[%d] = %f out of [0, Pi]", i, v)
		}
		if v := c.ZenithNeg[i]; v < 0 || v > math.Pi {
			t.Errorf("ZenithNeg[%d] = %f out of [0, Pi]", i, v)
		}
		if v := c.Polar[i]; v < 0 || v > math.Pi {
			t.Errorf("Polar[%d] = %f out of [0, Pi]", i, v)
		}
	}
}

func TestAt(t *testing.T) {
	c := Precompute(10)
	if got := c.At(3, 5); got != 53 {
		t.Errorf("At(3, 5) = %d, want 53", got)
	}
}

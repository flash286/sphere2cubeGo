package cache

import (
	"testing"
)

func TestPrecompute_Dimensions(t *testing.T) {
	sizes := []int{4, 16, 64}
	panoW, panoH := 200, 100
	for _, size := range sizes {
		pm := Precompute(size, panoW, panoH)
		n := size * size
		for face, table := range pm.Maps {
			if len(table) != n {
				t.Errorf("size=%d face=%d: expected %d elements, got %d", size, face, n, len(table))
			}
		}
	}
}

func TestPrecompute_OffsetsInRange(t *testing.T) {
	panoW, panoH := 200, 100
	size := 32
	pm := Precompute(size, panoW, panoH)
	maxOff := int32((panoW*panoH - 1) * 4)

	for face, table := range pm.Maps {
		for i, off := range table {
			if off < 0 || off > maxOff {
				t.Errorf("face=%d pixel=%d: offset %d out of range [0, %d]", face, i, off, maxOff)
			}
			if off%4 != 0 {
				t.Errorf("face=%d pixel=%d: offset %d not aligned to 4 bytes", face, i, off)
			}
		}
	}
}

func TestPrecompute_AllFaces(t *testing.T) {
	pm := Precompute(8, 100, 50)
	if pm.Size != 8 {
		t.Errorf("Size = %d, want 8", pm.Size)
	}
	for face, table := range pm.Maps {
		if len(table) != 64 {
			t.Errorf("face %d: len = %d, want 64", face, len(table))
		}
	}
}

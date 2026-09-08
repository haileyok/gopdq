package helpers

import (
	"image"
	"strings"
	"testing"
)

func TestHammingDistance(t *testing.T) {
	zeros := strings.Repeat("0", 64)
	ones := strings.Repeat("f", 64)
	cases := []struct {
		a, b     string
		expected int
	}{
		{zeros, zeros, 0},
		{ones, zeros, 256},
		{zeros[:62] + "80", zeros, 1},
	}
	for _, c := range cases {
		d, err := HammingDistance(c.a, c.b)
		if err != nil {
			t.Fatalf("HammingDistance(%s, %s): %v", c.a[:8], c.b[:8], err)
		}
		if d != c.expected {
			t.Errorf("HammingDistance(%s…, %s…) = %d, want %d", c.a[:8], c.b[:8], d, c.expected)
		}
	}
}

func TestHammingDistanceErrors(t *testing.T) {
	if _, err := HammingDistance("abc", "abc"); err == nil {
		t.Error("expected error for non-64-char hash")
	}
	if _, err := HammingDistance("0000000000000000000000000000000000000000000000000000000000000000", "000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected error for length mismatch")
	}
}

func TestPdqHashToBinary(t *testing.T) {
	b, err := PdqHashToBinary("00ff")
	if err != nil {
		t.Fatalf("PdqHashToBinary: %v", err)
	}
	if b != "0000000011111111" {
		t.Errorf("PdqHashToBinary(00ff) = %q, want 0000000011111111", b)
	}
}

func TestResizeIfNeeded(t *testing.T) {
	// Small image: returned unchanged (same instance is fine).
	small := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	if got := ResizeIfNeeded(small); got.Bounds().Dx() != 64 || got.Bounds().Dy() != 64 {
		t.Errorf("small image was resized: %v", got.Bounds())
	}
	// Large image: both dimensions clamped to <= 512.
	large := image.NewNRGBA(image.Rect(0, 0, 1024, 2048))
	got := ResizeIfNeeded(large)
	if got.Bounds().Dx() > 512 || got.Bounds().Dy() > 512 {
		t.Errorf("large image not clamped to 512: %v", got.Bounds())
	}
}

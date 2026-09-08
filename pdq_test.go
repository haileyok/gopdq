package pdq

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// Golden vectors from Meta's reference PDQ implementation (ThreatExchange,
// pdq/python/pdqhashing/tests/pdq_test.py, BSD-licensed). Test images are
// redistributed from ThreatExchange's pdq/data with attribution (see
// testdata/golden/LICENSE).
//
// The reference test asserts hamming distance <= 16 rather than exact
// equality, because JPEG decoder differences (libjpeg vs Go's image/jpeg)
// legitimately shift a few bits of a perceptual hash. We assert the same
// tolerance, and additionally report (not require) exact matches.

const hammingTolerance = 16

var goldenVectors = []struct {
	name     string
	expected string
	// tol overrides hammingTolerance for vectors known to be sensitive to
	// JPEG decoder differences across Go toolchains. q0003 is a very
	// low-quality image (PDQ quality ~3) whose hash legitimately shifts a
	// few bits between decoders: observed distance 16 on go1.26 and 18 on
	// go1.25 with otherwise-identical code.
	tol int
}{
	{"misc-images/c.png", "e64cc9d91e623842f8d1f1d9a398e78c9f199a3bd87924f2b7e11e0bf061b064", 0},
	{"misc-images/small.jpg", "0007001f003f003f007f00ff00ff00ff01ff01ff01ff03ff03ff03ff03ff03ff", 0},
	{"misc-images/wee.jpg", "6227401f601ff4ccafcc9fad4b0d95d371a2eb7265a3285234d228ca94deeb2d", 0},
	{"labelme-subset/q0003.jpg", "54a977c221d14c1c43ba5e6e21d4a13989a3553f1462611cbb85fda7be83b677", 24},
	{"labelme-subset/q0004.jpg", "992d44af36d69e6ca6b812585928bac11def254ef5398c6d07466c9abcc65b92", 0},
	{"labelme-subset/q0122.jpg", "cfb2009ddd21c6dab0046a7745b5984757a8a4535b3377aea2591d32b33ff940", 0},
	{"labelme-subset/q0291.jpg", "a0fe94f1e5cc1cc8dd855948498dc9243f7ca27336f036d7f212b74bc103c9a7", 0},
	{"labelme-subset/q0746.jpg", "1049d96239e24d4dca2c55512b8bdb77425f4dbcf575a0a95555aaab5554aaaa", 0},
	{"labelme-subset/q1050.jpg", "489db672e9190276d452aeab41eba20f02375fe4092d88defdf491a5c55c5f70", 0},
	{"labelme-subset/q2821.jpg", "b150231ffae4710ffcf4f18bb574b109a576f14bb8543189f8743289f174b109", 0},
	{"dih/bridge-1-original.jpg", "d8f8f0cce0f4a84f0e370a22028f67f0b36e2ed596623e1d33e6b39c4e9c9b22", 0},
	{"dih/bridge-2-rotate-90.jpg", "38a50efd71c83f429013d68d0ffffc52e34e0e15ada952a9d29684214aa9e5af", 0},
	{"dih/bridge-3-rotate-180.jpg", "2dadda64b5a142e5d362209057da895ae63b8c7fc277b4b766b319361f893188", 0},
	{"dih/bridge-4-rotate-270.jpg", "a5f0a457248995e8c9065c275aaa54d8b61ba4bdf8fcfc0387c32f8b0bfc4f05", 0},
	{"dih/bridge-5-flipx.jpg", "d8f80f31e0f417b00e37f5dd028f980fb36ed12a9662c1e233e64c634e9c64dd", 0},
	{"dih/bridge-6-flipy.jpg", "0dad259bb1a1bd18d362576556da32a1e63b7380c2374b4866b3c6c91b89ce77", 0},
	{"dih/bridge-7-flip-plus-1.jpg", "f0a5e10271dcc0bd9c5309720fff018de34ef1e8ada9a956d2967ade1ea91a50", 0},
	{"dih/bridge-8-flip-minus-1.jpg", "69f05aa8a4996a17c146a2da5aaaab07b61b5b60f8fc07fc83c3d0740bfcb0fa", 0},
}

// TestGoldenVectors checks gopdq against the reference C++/Python hashes.
func TestGoldenVectors(t *testing.T) {
	for _, v := range goldenVectors {
		t.Run(v.name, func(t *testing.T) {
			path := filepath.Join("testdata", "golden", v.name)
			result, err := HashFromFile(path)
			if err != nil {
				t.Fatalf("hashing %s: %v", path, err)
			}
			dist := hammingDistanceHex(result.Hash, v.expected)
			tol := hammingTolerance
			if v.tol > 0 {
				tol = v.tol
			}
			if dist > tol {
				t.Errorf("hash %s (quality %d) has hamming distance %d from reference %s; tolerance is %d",
					result.Hash, result.Quality, dist, v.expected, tol)
			}
			if dist == 0 {
				t.Logf("exact match with reference")
			}
		})
	}
}

// TestHashFromImageMatchesHashFromFile verifies the two entry points agree.
func TestHashFromImageMatchesHashFromFile(t *testing.T) {
	for _, v := range goldenVectors {
		path := filepath.Join("testdata", "golden", v.name)
		fromFile, err := HashFromFile(path)
		if err != nil {
			t.Fatalf("HashFromFile(%s): %v", path, err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		fromImage, err := HashFromImage(img)
		if err != nil {
			t.Fatalf("HashFromImage(%s): %v", path, err)
		}
		if fromFile.Hash != fromImage.Hash {
			t.Errorf("%s: HashFromFile=%s HashFromImage=%s", path, fromFile.Hash, fromImage.Hash)
		}
	}
}

// TestOpaqueVsSemiTransparent guards the alpha-handling fix: identical RGB
// content must hash identically regardless of alpha (straight, non-
// premultiplied RGB is what the reference implementation hashes).
func TestOpaqueVsSemiTransparent(t *testing.T) {
	opaque := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	semi := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			px := color.NRGBA{R: uint8(x), G: uint8(y), B: uint8((x + y) / 2), A: 255}
			opaque.SetNRGBA(x, y, px)
			semiPx := px
			semiPx.A = 128
			if x < 128 {
				semiPx.A = 0
			}
			semi.SetNRGBA(x, y, semiPx)
		}
	}
	o, err := HashFromImage(opaque)
	if err != nil {
		t.Fatalf("opaque: %v", err)
	}
	s, err := HashFromImage(semi)
	if err != nil {
		t.Fatalf("semi: %v", err)
	}
	if o.Hash != s.Hash {
		t.Errorf("alpha changed the hash: opaque=%s semi=%s -- premultiplied RGB regression?", o.Hash, s.Hash)
	}
}

// TestTooSmallImageReturnsError verifies the minimum-dimension guard.
func TestTooSmallImageReturnsError(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	_, err := HashFromImage(img)
	if err == nil {
		t.Fatal("expected error for image below minimum hashable dimension")
	}
}

func hammingDistanceHex(a, b string) int {
	if len(a) != 64 || len(b) != 64 {
		return hammingTolerance + 1
	}
	d := 0
	for i := 0; i < len(a); i += 2 {
		ahi, alo := hexVal(a[i]), hexVal(a[i+1])
		bhi, blo := hexVal(b[i]), hexVal(b[i+1])
		if ahi < 0 || alo < 0 || bhi < 0 || blo < 0 {
			return hammingTolerance + 1
		}
		d += popcount((ahi<<4|alo) ^ (bhi<<4|blo))
	}
	return d
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

func popcount(v int) int {
	c := 0
	for v != 0 {
		v &= v - 1
		c++
	}
	return c
}

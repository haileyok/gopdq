// Reimplementation of https://github.com/facebook/ThreatExchange/blob/main/pdq in Golang
//
// For reference, please see https://github.com/facebook/ThreatExchange/blob/main/hashing/hashing.pdf
//
// Function names are similar or the same as those in the reference C++ implementation, and
// any questions about implementation should reference that code.

package pdq

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// HashResult contains the output of a PDQ hash operation
type HashResult struct {
	Hash                  string
	Quality               int
	ImageHeightTimesWidth int
	HashDuration          time.Duration
}

// Various constants pulled from the reference implementation
const (
	LumaFromRCoeff = 0.299
	LumaFromGCoeff = 0.587
	LumaFromBCoeff = 0.114

	PdqNumJaroszXYPasses = 2

	DownsampleDims = 512

	MinHashableDim = 5
)

var (
	ErrInvalidFile = errors.New("invalid input file name")
)

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/pdqhashing.cpp#L42
var dctMatrix64 []float32

func init() {
	const numRows = 16
	const numCols = 64

	matrixScaleFactor := math.Sqrt(2.0 / float64(numCols))

	dctMatrix64 = make([]float32, numRows*numCols)

	for i := range numRows {
		for j := range numCols {
			dctMatrix64[i*numCols+j] = float32(matrixScaleFactor * math.Cos((math.Pi/2.0/float64(numCols))*float64(i+1)*float64(2*j+1)))
		}
	}
}

// HashFromImage generates a PDQ hash from an image.Image
// The image should idealy be pre-resizes to 512x512 or smaller for performance reasons.
// SEE: https://github.com/facebook/ThreatExchange/blob/main/hashing/hashing.pdf, "More on Downsampling"
// Returns a HashResult containing the hash and a quality score between 0 and 100.
// Please reference the evaluation data for selecting a good quality score. From hashing.pdf:
// "Confident-match distances are up to the system designer, of course, but 30, 20, or less has been found to
// produce good results on evaluation data."
func HashFromImage(img image.Image) (*HashResult, error) {
	bounds := img.Bounds()
	size := bounds.Size()

	imageHeightTimesWidth := size.Y * size.X

	luma, numRows, numCols := loadFloatLumaFromImage(img)

	fullBuffer2 := make([]float32, numRows*numCols)

	hashStart := time.Now()
	hash, quality := hash256FromFloatLuma(luma, fullBuffer2, numRows, numCols)
	hashTime := time.Since(hashStart)

	return &HashResult{
		Hash:                  hash,
		Quality:               quality,
		ImageHeightTimesWidth: imageHeightTimesWidth,
		HashDuration:          hashTime,
	}, nil
}

// Opens a file at the specified file and uses image.Image to decode the image. Returns the result of
// HashFromImage. This is a convenience wrapper around HashFromImage that handles the IO and decoding for you.
// Ideally, you should call HashFromImage on your own with a 512x512 or smaller image that you have resized
// yourself. This function is provided only to match the reference implementation.
func HashFromFile(filename string) (*HashResult, error) {
	if filename == "" {
		return nil, ErrInvalidFile
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return HashFromImage(img)
}

func loadFloatLumaFromImage(img image.Image) ([]float32, int, int) {
	bounds := img.Bounds()
	numRows := bounds.Dy()
	numCols := bounds.Dx()
	luma := make([]float32, numRows*numCols)

	for row := range numRows {
		for col := range numCols {
			// NOTE: color.Color.RGBA() returns alpha-premultiplied values; convert to
			// non-premultiplied NRGBA so that alpha is discarded without darkening the
			// RGB channels of semi-transparent pixels. See the reference C++ PDQ
			// implementation, which reads straight (non-premultiplied) RGB channels.
			c := color.NRGBAModel.Convert(img.At(bounds.Min.X+col, bounds.Min.Y+row)).(color.NRGBA)

			r8 := float32(c.R)
			g8 := float32(c.G)
			b8 := float32(c.B)

			luma[row*numCols+col] = LumaFromRCoeff*r8 + LumaFromGCoeff*g8 + LumaFromBCoeff*b8
		}
	}

	return luma, numRows, numCols
}

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/pdqhashing.cpp#L127
func hash256FromFloatLuma(
	fullBuffer1 []float32,
	fullBuffer2 []float32,
	numRows, numCols int,
) (string, int) {
	// from reference impl, do not return a hash for images taht are too small
	if numRows < MinHashableDim || numCols < MinHashableDim {
		return "", 0
	}

	buffer64x64 := make([]float32, 64*64)
	buffer16x64 := make([]float32, 16*64)
	buffer16x16 := make([]float32, 16*16)

	quality := float256FromFloatLuma(fullBuffer1, fullBuffer2, numRows, numCols, buffer64x64, buffer16x64, buffer16x16)

	hash := convertBufferToHash(buffer16x16)

	return hash, quality
}

const hexChars = "0123456789abcdef"

func convertBufferToHash(buffer16x16 []float32) string {
	median := torben(buffer16x16)

	words := make([]uint16, 16)

	for i := range 16 {
		for j := range 16 {
			if buffer16x16[i*16+j] > median {
				bitIndex := i*16 + j
				wordIndex := bitIndex / 16
				bitInWord := bitIndex % 16
				words[wordIndex] |= 1 << bitInWord
			}
		}
	}

	result := make([]byte, 64)
	for i := range 16 {
		word := words[15-i]
		offset := i * 4
		result[offset+0] = hexChars[word>>12]
		result[offset+1] = hexChars[(word>>8)&0xF]
		result[offset+2] = hexChars[(word>>4)&0xF]
		result[offset+3] = hexChars[word&0xF]
	}

	return string(result)
}

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/pdqhashing.cpp#L158
func float256FromFloatLuma(
	fullBuffer1 []float32,
	fullBuffer2 []float32,
	numRows, numCols int,
	buffer64x64 []float32,
	buffer16x64 []float32,
	buffer16x16 []float32,
) int {
	if numRows == 64 && numCols == 64 {
		copy(buffer64x64, fullBuffer1)
	} else {
		windowSizeAlongRows := computeJaroszFilterWindowSize(numCols, 64)
		windowSizeAlongCols := computeJaroszFilterWindowSize(numRows, 64)

		jaroszFilterFloat(fullBuffer1, fullBuffer2, numRows, numCols, windowSizeAlongRows, windowSizeAlongCols, PdqNumJaroszXYPasses)

		decimateFloat(fullBuffer1, numRows, numCols, buffer64x64, 64, 64)
	}

	quality := imageDomainQualityMetric(buffer64x64)

	dct64To16(buffer64x64, buffer16x64, buffer16x16)

	return quality
}

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/pdqhashing.cpp#L318
func imageDomainQualityMetric(buffer64x64 []float32) int {
	gradientSum := 0

	for i := range 63 {
		for j := range 64 {
			u := buffer64x64[i*64+j]
			v := buffer64x64[(i+1)*64+j]
			d := int(math.Abs(float64((u - v) * 100 / 255)))
			gradientSum += d
		}
	}

	for i := range 64 {
		for j := range 63 {
			u := buffer64x64[i*64+j]
			v := buffer64x64[i*64+j+1]
			d := int(math.Abs(float64((u - v) * 100 / 255)))
			gradientSum += d
		}
	}

	quality := min(gradientSum/90, 100)

	return quality
}

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/pdqhashing.cpp#L355
func dct64To16(A []float32, T []float32, B []float32) {
	for i := range 16 {
		dctRow := dctMatrix64[i*64:]
		for j := range 64 {
			var sum0, sum1, sum2, sum3 float32

			for k := 0; k < 64; k += 4 {
				sum0 += dctRow[k] * A[k*64+j]
				sum1 += dctRow[k+1] * A[(k+1)*64+j]
				sum2 += dctRow[k+2] * A[(k+2)*64+j]
				sum3 += dctRow[k+3] * A[(k+3)*64+j]
			}

			T[i*64+j] = sum0 + sum1 + sum2 + sum3
		}
	}

	for i := range 16 {
		tRow := T[i*64:]
		for j := range 16 {
			dctRow := dctMatrix64[j*64:]
			var sum0, sum1, sum2, sum3 float32

			for k := 0; k < 64; k += 4 {
				sum0 += tRow[k] * dctRow[k]
				sum1 += tRow[k+1] * dctRow[k+1]
				sum2 += tRow[k+2] * dctRow[k+2]
				sum3 += tRow[k+3] * dctRow[k+3]
			}

			B[i*16+j] = sum0 + sum1 + sum2 + sum3
		}
	}
}

// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/hashing/torben.cpp
func torben(m []float32) float32 {
	n := len(m)
	if n == 0 {
		return 0
	}

	min, max := m[0], m[0]
	for i := 1; i < n; i++ {
		if m[i] < min {
			min = m[i]
		}
		if m[i] > max {
			max = m[i]
		}
	}

	var guess, maxltguess, mingtguess float32
	var less, greater, equal int

	for {
		guess = (min + max) / 2
		less, greater, equal = 0, 0, 0
		maxltguess = min
		mingtguess = max

		for i := range n {
			if m[i] < guess {
				less++
				if m[i] > maxltguess {
					maxltguess = m[i]
				}
			} else if m[i] > guess {
				greater++
				if m[i] < mingtguess {
					mingtguess = m[i]
				}
			} else {
				equal++
			}
		}

		if less <= (n+1)/2 && greater <= (n+1)/2 {
			break
		} else if less > greater {
			max = maxltguess
		} else {
			min = mingtguess
		}
	}

	if less >= (n+1)/2 {
		return maxltguess
	} else if less+equal >= (n+1)/2 {
		return guess
	} else {
		return mingtguess
	}
}

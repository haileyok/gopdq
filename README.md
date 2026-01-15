# gopdq

A Go implementation of [Meta's PDQ](https://github.com/facebook/ThreatExchange/tree/main/pdq) perceptual hashing algorithm.

PDQ is a perceptual hashing algorithm designed to identify visually similar images. It generates a compact 256-bit hash that remains stable across common image transformations like resizing, compression, and minor edits.


## Installation

```bash
go get github.com/haileyok/gopdq
```

## Usage

There are two different functions provided in this package: `HashFromFile` and `HashFromImage`. While either will work, you should ensure that the input image has been resized to a size no greater than 512x512. See
[the PDQ paper](https://github.com/facebook/ThreatExchange/blob/main/hashing/hashing.pdf).

> Using two-pass Jarosz filters (i.e. tent convolutions), compute a weighted average of 64x64 subblocks of
the luminance image. (This is prohibitively time-consuming for megapixel input so we recommend using an
off-the-shelf technique to first resize to 512x512 before converting from RGB to luminance.)

For conveneicne, there is a helper method `helpers.ResizeIfNeeded(img image.Image)` which will return a resized `image.Image` that can be passed to `HashFromImage`.


```go
package main

import (
    "fmt"
    "log"

    "github.com/haileyok/gopdq"
)

func main() {
    // Hash an image file, assuming it has already been resized.
    // NOTE: There is no logic that _guarantees_ an image has been resized, this is up to you to ensure.
    result, err := pdq.HashFromFile("image.jpg")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Hash: %s\n", result.Hash)
    fmt.Printf("Quality: %d\n", result.Quality)
}
```

### Using with pre-loaded images

```go
import (
    "image"
    _ "image/jpeg"

    "github.com/haileyok/gopdq"
    "github.com/haileyok/gopdq/helpers"
)

func main() {
    // Open the image and decode it
    file, _ := os.Open("image.jpg")
    img, _, _ := image.Decode(file)

    // Resize if needed
    img = helpers.ResizeIfNeeded(img)

    // Generate hash
    result, _ := pdq.HashFromImage(img)
    fmt.Println(result.Hash)
}
```

### HashResult

Both of the above functions will return a `HashResult`, which includes both the hash and the quality score.

```go
type HashResult struct {
    Hash                  string
    Quality               int           // Results with a quality score < 50 should be discarded
    ImageHeightTimesWidth int
    HashDuration          time.Duration
}
```

## Command Line Tools

### PDQ Hasher

```bash
# Build the hasher
go build ./cmd/pdqhasher

# Hash an image
./pdqhasher path/to/image.jpg

# Output:
# Hash: e77b19ca5399466258c656bc4666a7853939a567a9193939e667199856ccc6c6
# Quality: 100
# Binary: 1110011110110001000110011010010100110011100110010100011001100010...
```

### Hamming Distance Helper

```bash
# Build the helper
go build ./cmd/helper

# Calculate hamming distance
./helper hamming <hash1> <hash2>

# Output:
# 8
```

## About Distance

Please see https://github.com/facebook/ThreatExchange/tree/main/pdq#matching

Note that outputs from the C++ implementation's example binary and the `pdqhasher` binary provided here may not return hashes that are exactly the same due to
differences in resizing libraries. This is expected, see https://github.com/facebook/ThreatExchange/tree/main/pdq#hashing.

## Benchmark

```
❯ go run ./cmd/benchmark --workers 32 --with-resize --duration 10
CPU:             AMD RYZEN AI MAX+ 395 w/ Radeon 8060S
CPU Cores:       32
Image Directory: testdata/images
Duration:        10s
Workers:         32
With Resize:     true
With I/O:        false

Results
=======

Total Time:       10.011804696s
Total Hashes:     27999
Errors:           0

Throughput:       2796.6 hashes/sec
Avg Time/Hash:    0.36 ms

Per Worker:       875.0 hashes
Per Worker/Sec:   87.4 hashes/sec
```

## References

- [PDQ Algorithm (C++ Reference)](https://github.com/facebook/ThreatExchange/tree/main/pdq)
- [PDQ Hashing Paper](https://github.com/facebook/ThreatExchange/blob/main/hashing/hashing.pdf)
- [ThreatExchange](https://github.com/facebook/ThreatExchange)

## Acknowledgments

This is a Go implementation of Meta's PDQ algorithm. All credit for the algorithm design goes to the original authors.

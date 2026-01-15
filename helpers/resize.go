package helpers

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	pdq "github.com/haileyok/gopdq"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func ResizeIfNeeded(img image.Image) image.Image {
	size := img.Bounds().Size()

	if size.X > pdq.DownsampleDims || size.Y > pdq.DownsampleDims {
		// SEE: https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/io/pdqio.cpp#L103
		// we use NearestNeighbor here as the PDQ uses that algo as well (unspecified parameter
		// which defaults to nearest neighbor, see https://cimg.eu/reference/structcimg__library_1_1CImg.html)
		// even still, because the two libraries have different implementations, we'll still see
		// minor differences in output. that is expected. see "More on Downsampling" in hashing.pdf
		return resize.Resize(pdq.DownsampleDims, pdq.DownsampleDims, img, resize.NearestNeighbor)
	}

	return img
}

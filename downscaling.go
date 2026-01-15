// Reimplementation of https://github.com/facebook/ThreatExchange/blob/main/pdq in Golang
//
// For reference, please see https://github.com/facebook/ThreatExchange/blob/main/hashing/hashing.pdf
//
// Function names are similar or the same as those in the reference C++ implementation, and
// any questions about implementation should reference that code.
//
// Downscaling reference found at https://github.com/facebook/ThreatExchange/blob/main/pdq/cpp/downscaling/downscaling.cpp

package pdq

func computeJaroszFilterWindowSize(oldDimension, newDimension int) int {
	return (oldDimension + 2*newDimension - 1) / (2 * newDimension)
}

func jaroszFilterFloat(buffer1 []float32, buffer2 []float32, numRows, numCols, windowSizeAlongRows, windowSizeAlongCols, nreps int) {
	for range nreps {
		boxAlongRowsFloat(buffer1, buffer2, numRows, numCols, windowSizeAlongRows)
		boxAlongColsFloat(buffer2, buffer1, numRows, numCols, windowSizeAlongCols)
	}
}

func boxAlongRowsFloat(inVector []float32, outVector []float32, numRows, numCols, windowSize int) {
	for i := range numRows {
		offset := i * numCols
		box1DFloat(inVector[offset:], outVector[offset:], numCols, 1, windowSize)
	}
}

func boxAlongColsFloat(inVector []float32, outVector []float32, numRows, numCols, windowSize int) {
	for j := range numCols {
		box1DFloat(inVector[j:], outVector[j:], numRows, numCols, windowSize)
	}
}

func box1DFloat(inVector []float32, outVector []float32, vectorLength, stride, fullWindowSize int) {
	halfWindowSize := (fullWindowSize + 2) / 2

	phase1Nreps := halfWindowSize - 1
	phase2Nreps := fullWindowSize - halfWindowSize + 1
	phase3Nreps := vectorLength - fullWindowSize
	phase4Nreps := halfWindowSize - 1

	var li, ri, oi int
	var sum float32
	var currentWindowSize int

	for range phase1Nreps {
		sum += inVector[ri]
		currentWindowSize++
		ri += stride
	}

	for range phase2Nreps {
		sum += inVector[ri]
		currentWindowSize++
		outVector[oi] = sum / float32(currentWindowSize)
		ri += stride
		oi += stride
	}

	invWindowSize := 1.0 / float32(currentWindowSize)
	for range phase3Nreps {
		sum += inVector[ri]
		sum -= inVector[li]
		outVector[oi] = sum * invWindowSize
		li += stride
		ri += stride
		oi += stride
	}

	for range phase4Nreps {
		sum -= inVector[li]
		currentWindowSize--
		outVector[oi] = sum / float32(currentWindowSize)
		li += stride
		oi += stride
	}
}

func decimateFloat(in []float32, inNumRows, inNumCols int, out []float32, outNumRows, outNumCols int) {
	// target centers not corners:
	for outRow := range outNumRows {
		inRow := int(((float32(outRow) + 0.5) * float32(inNumRows)) / float32(outNumRows))
		for outCol := range outNumCols {
			inCol := int(((float32(outCol) + 0.5) * float32(inNumCols)) / float32(outNumCols))
			out[outRow*outNumCols+outCol] = in[inRow*inNumCols+inCol]
		}
	}
}

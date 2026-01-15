// slop code that seems to work fine

package main

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "net/http/pprof"

	pdq "github.com/haileyok/gopdq"
	"github.com/haileyok/gopdq/helpers"
	"github.com/urfave/cli/v3"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func main() {
	app := cli.Command{
		Name:  "benchmark",
		Usage: "Measure PDQ hashing throughput",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Value:   "testdata/images",
				Usage:   "Directory containing test images",
			},
			&cli.IntFlag{
				Name:    "duration",
				Aliases: []string{"t"},
				Value:   10,
				Usage:   "Duration in seconds to run the benchmark",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Value:   1,
				Usage:   "Number of parallel workers (0 = auto = num CPUs)",
			},
			&cli.BoolFlag{
				Name:    "with-resize",
				Aliases: []string{"r"},
				Value:   true,
				Usage:   "Include resize in benchmark",
			},
			&cli.BoolFlag{
				Name:    "with-io",
				Aliases: []string{"i"},
				Value:   false,
				Usage:   "Include file I/O in benchmark (slower)",
			},
			&cli.StringFlag{
				Name:  "metrics-addr",
				Value: ":6009",
				Usage: "Address for pprof server",
			},
		},
		Action: runBenchmark,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runBenchmark(ctx context.Context, cmd *cli.Command) error {
	metricsServer := http.DefaultServeMux
	go func() {
		if err := http.ListenAndServe(cmd.String("metrics-addr"), metricsServer); err != nil {
			log.Fatal(err)
		}
	}()

	imageDir := cmd.String("dir")
	duration := time.Duration(cmd.Int("duration")) * time.Second
	numWorkers := cmd.Int("workers")
	withResize := cmd.Bool("with-resize")
	withIO := cmd.Bool("with-io")

	if numWorkers == 0 {
		numWorkers = 8
	}

	cpuInfo := getCPUInfo()

	fmt.Printf("PDQ Hashing Throughput Benchmark\n")
	fmt.Printf("=================================\n\n")
	fmt.Printf("CPU:             %s\n", cpuInfo)
	fmt.Printf("CPU Cores:       %d\n", runtime.NumCPU())
	fmt.Printf("Image Directory: %s\n", imageDir)
	fmt.Printf("Duration:        %v\n", duration)
	fmt.Printf("Workers:         %d\n", numWorkers)
	fmt.Printf("With Resize:     %v\n", withResize)
	fmt.Printf("With I/O:        %v\n\n", withIO)

	type imageWithSize struct {
		img          image.Image
		originalSize int // max dimension before resize
	}

	var imagePaths []string
	var preloadedImages []imageWithSize

	fmt.Print("Loading test images... ")
	if withIO {
		var err error
		imagePaths, err = loadImagePaths(imageDir)
		if err != nil {
			return fmt.Errorf("failed to load images: %w", err)
		}
		fmt.Printf("%d images found\n\n", len(imagePaths))
	} else {
		var err error
		imagePaths, err = loadImagePaths(imageDir)
		if err != nil {
			return fmt.Errorf("failed to load images: %w", err)
		}

		for _, path := range imagePaths {
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			img, _, err := image.Decode(file)
			file.Close()
			if err != nil {
				continue
			}

			// Store original size before resize
			bounds := img.Bounds()
			originalSize := max(bounds.Dx(), bounds.Dy())

			if withResize {
				img = helpers.ResizeIfNeeded(img)
			}

			preloadedImages = append(preloadedImages, imageWithSize{
				img:          img,
				originalSize: originalSize,
			})
		}
		fmt.Printf("%d images loaded and decoded\n\n", len(preloadedImages))
	}

	if len(imagePaths) == 0 && len(preloadedImages) == 0 {
		return fmt.Errorf("no images found in %s. Run setup_testdata.py to download test images", imageDir)
	}

	// Run benchmark
	fmt.Println("Starting benchmark...")
	fmt.Println()

	var hashCount atomic.Int64
	var errorCount atomic.Int64
	startTime := time.Now()
	stopTime := startTime.Add(duration)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			idx := 0
			for time.Now().Before(stopTime) {
				var err error

				if withIO {
					// Hash from file
					imagePath := imagePaths[idx%len(imagePaths)]
					_, err = pdq.HashFromFile(imagePath)
				} else {
					// Hash from pre-decoded image
					imgWithSize := preloadedImages[idx%len(preloadedImages)]
					_, err = pdq.HashFromImage(imgWithSize.img)
				}

				if err != nil {
					errorCount.Add(1)
				} else {
					hashCount.Add(1)
				}
				idx++
			}
		}(w)
	}

	// Progress reporting
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for time.Now().Before(stopTime) {
			<-ticker.C
			elapsed := time.Since(startTime)
			count := hashCount.Load()
			rate := float64(count) / elapsed.Seconds()
			fmt.Printf("\rElapsed: %5.1fs | Hashes: %7d | Rate: %8.1f hashes/sec", elapsed.Seconds(), count, rate)
		}
	}()

	// Wait for workers
	wg.Wait()

	elapsed := time.Since(startTime)
	totalHashes := hashCount.Load()
	totalErrors := errorCount.Load()
	hashesPerSecond := float64(totalHashes) / elapsed.Seconds()

	// Final results
	fmt.Printf("\n\n")
	fmt.Printf("Results\n")
	fmt.Printf("=======\n\n")
	fmt.Printf("Total Time:       %v\n", elapsed)
	fmt.Printf("Total Hashes:     %d\n", totalHashes)
	fmt.Printf("Errors:           %d\n", totalErrors)
	fmt.Printf("\n")
	fmt.Printf("Throughput:       %.1f hashes/sec\n", hashesPerSecond)
	fmt.Printf("Avg Time/Hash:    %.2f ms\n", 1000.0/hashesPerSecond)
	fmt.Printf("\n")

	// Per-worker stats
	hashesPerWorker := float64(totalHashes) / float64(numWorkers)
	fmt.Printf("Per Worker:       %.1f hashes\n", hashesPerWorker)
	fmt.Printf("Per Worker/Sec:   %.1f hashes/sec\n", hashesPerWorker/elapsed.Seconds())

	// Size breakdown if we have preloaded images
	if !withIO && len(preloadedImages) > 0 {
		fmt.Printf("\n")
		fmt.Printf("Image Size Breakdown (Original Sizes)\n")
		fmt.Printf("======================================\n\n")

		var small, medium, large int
		for _, imgWithSize := range preloadedImages {
			maxDim := imgWithSize.originalSize
			if maxDim <= 512 {
				small++
			} else if maxDim <= 1024 {
				medium++
			} else {
				large++
			}
		}

		fmt.Printf("Small (≤512):      %d (%.1f%%)\n", small, float64(small)/float64(len(preloadedImages))*100)
		fmt.Printf("Medium (513-1024): %d (%.1f%%)\n", medium, float64(medium)/float64(len(preloadedImages))*100)
		fmt.Printf("Large (>1024):     %d (%.1f%%)\n", large, float64(large)/float64(len(preloadedImages))*100)
	}

	return nil
}

func loadImagePaths(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp":
			images = append(images, path)
		}
		return nil
	})

	return images, err
}

func getCPUInfo() string {
	// Try to read CPU info from /proc/cpuinfo (Linux)
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Fallback to architecture info
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

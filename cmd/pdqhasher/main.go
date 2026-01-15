package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"time"

	pdq "github.com/haileyok/gopdq"
	"github.com/haileyok/gopdq/helpers"
	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name: "pdqhasher",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "input-file",
				UsageText: "path to input file to get a hash from",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fileName := cmd.StringArg("input-file")

			readStart := time.Now()
			file, err := os.Open(fileName)
			if err != nil {
				return fmt.Errorf("failed to open file at %s: %w", fileName, err)
			}
			defer file.Close()

			// decode the image so we can resize if needed
			img, _, err := image.Decode(file)

			readDuration := time.Since(readStart)

			resizeStart := time.Now()
			// resize the image if needed, since the implementation does not do resizing for you
			img = helpers.ResizeIfNeeded(img)
			resizeDuration := time.Since(resizeStart)

			// create a hash from the image
			hash, err := pdq.HashFromImage(img)
			if err != nil {
				return fmt.Errorf("failed to hash input image: %w", err)
			}

			binary, _ := helpers.PdqHashToBinary(hash.Hash)

			// return the hash and the quality
			fmt.Printf("\nHash: %s\nQuality: %d\nBinary: %s\n\nRead Microseconds: %d\nResize Microseconds: %d\nHash Microseconds: %d\n", hash.Hash, hash.Quality, binary, readDuration.Microseconds(), resizeDuration.Microseconds(), hash.HashDuration.Microseconds())

			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

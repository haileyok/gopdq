package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/haileyok/gopdq/helpers"
	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name: "pdq-helper",
		Commands: []*cli.Command{
			{
				Name: "hamming",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "hash-1",
					},
					&cli.StringArg{
						Name: "hash-2",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					hashOne := cmd.StringArg("hash-1")
					hashTwo := cmd.StringArg("hash-2")

					distance, err := helpers.HammingDistance(hashOne, hashTwo)
					if err != nil {
						return fmt.Errorf("failed to get distance between two input hashes: %w", err)
					}

					fmt.Println(distance)

					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

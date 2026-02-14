package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	cmd := buildCLI()

	args := os.Args
	if len(args) == 1 {
		args = append(args, "start")
	}

	if err := cmd.Run(context.Background(), args); err != nil {
		slog.Error("application exited", slog.Any("error", err))
		os.Exit(1)
	}
}

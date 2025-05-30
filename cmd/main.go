package main

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/cmd/redditpostnotifier"
	"log"
	"os"
)

func main() {
	app := redditpostnotifier.BuildCLI()

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "start")
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

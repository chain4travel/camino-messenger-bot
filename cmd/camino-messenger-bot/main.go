package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/app"
)

var (
	// these variables are set by go build -ldflags
	// TODO: @VjeraTurk make this work when multiple bots are ran with launch.json
	Version   string = "unknown"
	GitCommit string = "unknown"
)

func main() {
	fmt.Println("Version\t", Version)
	fmt.Println("GitCommit\t", GitCommit)

	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := app.NewApp(cfg)
	if err != nil {
		panic(err)
	}
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}

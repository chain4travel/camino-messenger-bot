package main

import (
	"context"
	"os/signal"
	"syscall"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/app"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := app.NewApp(cfg)
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}

}

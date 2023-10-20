package main

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/app"
	"context"
	"os/signal"
	"syscall"
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

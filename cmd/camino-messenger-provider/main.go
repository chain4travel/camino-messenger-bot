package main

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/app"
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	time.Sleep(8 * time.Second)
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := app.NewApp(cfg)
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}

}

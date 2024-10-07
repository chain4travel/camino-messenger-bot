package main

import (
	"context"
	"os/signal"
	"syscall"

	"log"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// these variables are set by go build -ldflags
	// TODO: @VjeraTurk make this work when multiple bots are ran with launch.json
	Version   string
	GitCommit string
)

var rootCmd = &cobra.Command{
	Use:        "camino-messenger-bot",
	Short:      "starts camino messenger bot",
	Version:    Version,
	SuggestFor: []string{"camino-messenger", "camino-messenger-bot", "camino-bot", "cmb"},
	RunE:       rootFunc,
}

func rootFunc(cmd *cobra.Command, _ []string) error {
	// TODO@
	log.Printf("Version %s", Version)
	log.Printf("GitCommit %s", GitCommit)

	isDevelopmentMode, err := cmd.Flags().GetBool(config.FlagKeyDeveloperMode)
	if err != nil {
		log.Fatalf("failed to get developer mode flag: %v", err)
	}

	var zapLogger *zap.Logger
	if isDevelopmentMode {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	logger := zapLogger.Sugar()
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.ReadConfig(logger)
	if err != nil {
		logger.Error(err)
		return err
	}

	app, err := app.NewApp(ctx, cfg, logger)
	if err != nil {
		logger.Error(err)
		return err
	}

	return app.Run(ctx)
}

func init() {
	cobra.EnablePrefixMatching = true

	if err := config.BindFlags(rootCmd); err != nil {
		log.Fatalf("failed to bind flags: %v", err)
	}
}

package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

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
	configReader := config.NewConfigReader(cmd.Flags())

	var err error
	var zapLogger *zap.Logger
	if configReader.IsDevelopmentMode() {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	logger := zapLogger.Sugar()
	defer func() { _ = logger.Sync() }()

	logger.Infof("App version: %s (git: %s)", Version, GitCommit)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := configReader.ReadConfig(logger)
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
	rootCmd.Flags().AddFlagSet(config.Flags())
}

func Execute() error {
	return rootCmd.Execute()
}
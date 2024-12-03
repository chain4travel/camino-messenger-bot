package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/app"
	"github.com/chain4travel/camino-messenger-bot/internal/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:        "camino-messenger-bot",
	Short:      "starts camino messenger bot",
	Version:    version.AppVersion,
	SuggestFor: []string{"camino-messenger", "camino-messenger-bot", "camino-bot", "cmb"},
	RunE:       rootFunc,
}

func rootFunc(cmd *cobra.Command, _ []string) error {
	configReaderLogger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to create config-reader logger: %w", err)
	}

	sugaredConfigReaderLogger := configReaderLogger.Sugar()
	defer func() { _ = sugaredConfigReaderLogger.Sync() }()

	configReader, err := config.NewConfigReader(cmd.Flags(), sugaredConfigReaderLogger)
	if err != nil {
		return fmt.Errorf("failed to create config reader: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := configReader.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	_ = sugaredConfigReaderLogger.Sync()

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

	logger.Infof("App version: %s (git: %s)", version.AppVersion, version.AppGitCommit)
	logger.Infof("Protocol version: %s", version.ProtocolVersion)
	logger.Infof("buf.build protocolbuffers version: %s", version.BufBuildPBCommit)
	logger.Infof("buf.build grpc version: %s", version.BufBuildGRPCCommit)
	logger.Infof("camino-messenger-contracts version: %s", version.ContractsGitCommit)

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

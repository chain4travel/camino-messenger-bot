// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/app"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var rootCmd = &cobra.Command{
	Use:           "camino-messenger-bot",
	Short:         "starts camino messenger bot",
	Version:       version.FullVersion,
	SuggestFor:    []string{"camino-messenger", "camino-messenger-bot", "camino-bot", "cmb"},
	RunE:          rootFunc,
	SilenceErrors: true,
	SilenceUsage:  true,
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

	var zapLoggerConfig zap.Config
	var zapLoggerOptions []zap.Option
	if configReader.IsDevelopmentMode() {
		zapLoggerConfig = zap.NewDevelopmentConfig()
		zapLoggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapLoggerOptions = append(zapLoggerOptions, zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		zapLoggerConfig = zap.NewProductionConfig()
	}
	zapLoggerConfig.OutputPaths = []string{"stdout"}
	zapLoggerConfig.ErrorOutputPaths = []string{"stderr"}
	zapLogger, err := zapLoggerConfig.Build(zapLoggerOptions...)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	logger := zapLogger.Sugar()
	defer func() { _ = logger.Sync() }()

	if err := version.InitProtoVersion(logger, configReader.IsDevelopmentMode()); err != nil {
		err := fmt.Errorf("failed to initialize bot version: %w", err)
		logger.Error(err)
		return err
	}

	logger.Infof("App version: %s (git: %s)", version.AppVersion, version.AppGitCommit)
	logger.Infof("buf.build protocolbuffers version: %s (CMP %s)", version.BufBuildPBCommit, version.BufBuildPBCMPRelease)
	logger.Infof("buf.build grpc version: %s (CMP %s)", version.BufBuildGRPCCommit, version.BufBuildGRPCCMPRelease)
	logger.Infof("camino-messenger-contracts version: %s", version.ContractsGitCommit)

	app, err := app.NewApp(ctx, cfg, logger)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// if context is canceled, it means that the app was stopped by a signal
			return nil
		}

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

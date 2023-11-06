package app

import (
	"context"
	"time"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/matrix"
	"camino-messenger-bot/internal/messaging"
	"camino-messenger-bot/internal/rpc/client"
	"camino-messenger-bot/internal/rpc/server"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg    *config.Config
	logger *zap.SugaredLogger
}

func NewApp(cfg *config.Config) (*App, error) {
	app := &App{
		cfg: cfg,
	}

	// create logger
	var logger *zap.Logger
	// Development configuration with a lower log level (e.g., Debug).
	if cfg.DeveloperMode {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, _ = cfg.Build()
	} else { // Production configuration with a higher log level (e.g., Info).
		cfg := zap.NewProductionConfig()
		logger, _ = cfg.Build()
	}
	app.logger = logger.Sugar()
	defer logger.Sync()

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	//// TODO do proper DI with FX

	rpcClient := client.NewClient(&a.cfg.PartnerPluginConfig, a.logger)
	g.Go(func() error {
		a.logger.Info("Starting RPC client...")
		return rpcClient.Start()
	})

	messenger := matrix.NewMessenger(&a.cfg.MatrixConfig, a.logger)
	userIDUpdated := make(chan string) // Channel to pass the userID
	g.Go(func() error {
		a.logger.Info("Starting message receiver...")
		userID, err := messenger.StartReceiver()
		// Pass userID through the channel
		userIDUpdated <- userID
		return err
	})

	msgProcessor := messaging.NewProcessor(messenger, rpcClient, a.logger, time.Duration(a.cfg.MessengerConfig.Timeout)*time.Millisecond)
	g.Go(func() error {
		// Wait for userID to be passed
		userID := <-userIDUpdated
		close(userIDUpdated)
		msgProcessor.SetUserID(userID)
		a.logger.Info("Starting message processor...")
		msgProcessor.Start(ctx)
		return nil
	})

	rpcServer := server.NewServer(&a.cfg.RPCServerConfig, a.logger, []grpc.ServerOption{}, msgProcessor) //TODO
	g.Go(func() error {
		a.logger.Info("Starting RPC server...")
		rpcServer.Start()
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		return messenger.StopReceiver()
	})

	g.Go(func() error {
		<-gCtx.Done()
		rpcServer.Stop()
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		return rpcClient.Shutdown()
	})

	if err := g.Wait(); err != nil {
		a.logger.Error(err)
	}
	return nil
}

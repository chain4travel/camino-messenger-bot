package app

import (
	"context"
	rsa_util "github.com/chain4travel/camino-messenger-bot/utils/rsa"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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
	serviceRegistry := messaging.NewServiceRegistry(a.logger, rpcClient)
	g.Go(func() error {
		a.logger.Info("Starting gRPC client...")
		err := rpcClient.Start()
		if err != nil {
			panic(err)
		}
		serviceRegistry.RegisterServices(a.cfg.SupportedRequestTypes)
		return nil
	})

	if a.cfg.BotMode > 2 {
		a.logger.Error("Invalid bot mode")
		return nil
	}
	privateRSAKey, err := rsa_util.ParseRSAPrivateKeyFromFile(a.cfg.PrivateRSAFileKey)
	if err != nil {
		a.logger.Error("Error while parsing private RSA key")
		return nil
	}
	messenger := matrix.NewMessenger(&a.cfg.MatrixConfig, a.logger, privateRSAKey)
	userIDUpdated := make(chan string) // Channel to pass the userID
	g.Go(func() error {
		a.logger.Infof("Starting message receiver with botmode %d ...", a.cfg.BotMode)
		userID, err := messenger.StartReceiver(a.cfg.BotMode)
		if err != nil {
			panic(err)
		}
		userIDUpdated <- userID // Pass userID through the channel
		return nil
	})

	msgProcessor := messaging.NewProcessor(messenger, a.logger, a.cfg.ProcessorConfig, serviceRegistry)
	g.Go(func() error {
		// Wait for userID to be passed
		userID := <-userIDUpdated
		close(userIDUpdated)
		msgProcessor.SetUserID(userID)
		a.logger.Info("Starting message processor...")
		msgProcessor.Start(ctx)
		return nil
	})

	rpcServer := server.NewServer(&a.cfg.RPCServerConfig, a.logger, msgProcessor, serviceRegistry)
	g.Go(func() error {
		a.logger.Info("Starting gRPC server...")
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

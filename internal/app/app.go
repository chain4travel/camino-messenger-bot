package app

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/matrix"
	"camino-messenger-provider/internal/messaging"
	"camino-messenger-provider/internal/rpc/client"
	"camino-messenger-provider/internal/rpc/server"
	"context"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg          *config.Config
	logger       *zap.SugaredLogger
	matrixClient matrix.Client
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

	// create matrix client
	app.matrixClient = matrix.NewClient(cfg.MatrixHost, app.logger, matrix.DefaultInterval)
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	// login
	err := a.matrixClient.Login(matrix.LoginRequest{
		Type: "m.login.password",
		Identifier: matrix.LoginRequestIdentifier{
			Type:    "m.id.thirdparty",
			Medium:  matrix.ThirdPartyIdentifierMedium,
			Address: a.cfg.Username,
		},
		Password: a.cfg.Password,
	})
	if err != nil {
		return err
	}

	rpcServer := server.NewServer(&a.cfg.RPCServerConfig, a.logger, []grpc.ServerOption{}) //TODO
	g.Go(func() error {
		a.logger.Info("Starting RPC server...")
		rpcServer.Start()
		return nil
	})

	rpcClient := client.NewClient(&a.cfg.PartnerPluginConfig, a.logger)
	g.Go(func() error {
		a.logger.Info("Starting RPC client...")
		return rpcClient.Start()
	})

	msgProcessor := messaging.NewProcessor(a.matrixClient, rpcClient, a.cfg.Username, a.logger)
	g.Go(func() error {
		a.logger.Info("Starting message processor...")
		msgProcessor.Start(ctx)
		return nil
	})

	poller := messaging.NewPoller(a.matrixClient, a.logger, msgProcessor)
	g.Go(func() error {
		a.logger.Info("Starting message receiver...")
		return poller.Start()
	})

	g.Go(func() error {
		<-gCtx.Done()
		poller.Stop()
		return nil
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

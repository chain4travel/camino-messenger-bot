package app

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/matrix"
	"camino-messenger-provider/internal/messaging"
	"camino-messenger-provider/internal/rpc/server"
	"context"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"log"
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
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
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

	messaging.NewPoller(a.matrixClient, a.logger, a.cfg.Username)
	//g.Go(func() error {
	//	a.logger.Info("Starting message receiver...")
	//	return poller.Start()
	//})

	rpcServer := server.NewServer([]grpc.ServerOption{}) //TODO
	g.Go(func() error {
		a.logger.Info("Starting RPC server...")
		rpcServer.Start()
		return nil
	})

	//g.Go(func() error {
	//	<-gCtx.Done()
	//	poller.Stop()
	//	return nil
	//})

	g.Go(func() error {
		<-gCtx.Done()
		rpcServer.Stop()
		return nil
	})
	if err := g.Wait(); err != nil {
		a.logger.Error(err)
	}
	return nil
}

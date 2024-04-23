package app

import (
	"context"

	"fmt"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/internal/tvm"
	"github.com/chain4travel/camino-messenger-bot/utils/constants"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg    *config.Config
	logger *zap.SugaredLogger
	tracer tracing.Tracer
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

	// TODO do proper DI with FX

	serviceRegistry := messaging.NewServiceRegistry(a.logger)
	// start rpc client if host is provided, otherwise bot serves as a distributor bot (rpc server)
	if a.cfg.PartnerPluginConfig.Host != "" {
		a.startRPCClient(gCtx, g, *serviceRegistry)
	} else {
		a.logger.Infof("No host for partner plugin provided, bot will serve as a distributor bot.")
		serviceRegistry.RegisterServices(a.cfg.SupportedRequestTypes, nil)
	}
	// start messenger (receiver)
	messenger, userIDUpdatedChan := a.startMessenger(gCtx, g)

	// initiate tvm client
	responseHandler := a.initTVMClient()

	// start msg processor
	msgProcessor := a.startMessageProcessor(ctx, messenger, serviceRegistry, responseHandler, g, userIDUpdatedChan)

	// init tracer
	tracer := a.initTracer()
	defer func() {
		if err := tracer.Shutdown(); err != nil {
			a.logger.Fatal("failed to shutdown tracer: %w", err)
		}
	}()

	// start rpc server
	a.startRPCServer(gCtx, msgProcessor, serviceRegistry, g)

	if err := g.Wait(); err != nil {
		a.logger.Error(err)
	}
	return nil
}

func (a *App) initTracer() tracing.Tracer {
	var (
		tracer tracing.Tracer
		err    error
	)
	if a.cfg.TracingConfig.Enabled {
		tracer, err = tracing.NewTracer(&a.cfg.TracingConfig, fmt.Sprintf("%s:%d", constants.AppName, a.cfg.RPCServerConfig.Port))
	} else {
		tracer, err = tracing.NewNoOpTracer()
	}
	if err != nil {
		a.logger.Fatal("failed to initialize tracer: %w", err)
	}
	a.tracer = tracer
	return tracer
}

func (a *App) initTVMClient() messaging.ResponseHandler {
	var responseHandler messaging.ResponseHandler
	// TODO make client init conditional based on provided config
	tvmClient, err := tvm.NewClient(a.cfg.TvmConfig)
	if err != nil {
		// do no return error here, let the bot continue
		a.logger.Warnf("Failed to create tvm client: %v", err)
		responseHandler = messaging.NoopResponseHandler{}
	} else {
		responseHandler = messaging.NewResponseHandler(tvmClient, a.logger)
	}
	return responseHandler
}

func (a *App) startRPCClient(ctx context.Context, g *errgroup.Group, serviceRegistry messaging.ServiceRegistry) {
	rpcClient := client.NewClient(&a.cfg.PartnerPluginConfig, a.logger)
	g.Go(func() error {
		a.logger.Info("Starting gRPC client...")
		err := rpcClient.Start()
		if err != nil {
			panic(err)
		}
		serviceRegistry.RegisterServices(a.cfg.SupportedRequestTypes, rpcClient)
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		return rpcClient.Shutdown()
	})
}

func (a *App) startMessenger(ctx context.Context, g *errgroup.Group) (messaging.Messenger, chan string) {
	messenger := matrix.NewMessenger(&a.cfg.MatrixConfig, a.logger)
	userIDUpdatedChan := make(chan string) // Channel to pass the userID
	g.Go(func() error {
		a.logger.Info("Starting message receiver...")
		userID, err := messenger.StartReceiver()
		if err != nil {
			panic(err)
		}
		userIDUpdatedChan <- userID // Pass userID through the channel
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		return messenger.StopReceiver()
	})
	return messenger, userIDUpdatedChan
}

func (a *App) startRPCServer(ctx context.Context, msgProcessor messaging.Processor, serviceRegistry *messaging.ServiceRegistry, g *errgroup.Group) {
	rpcServer := server.NewServer(&a.cfg.RPCServerConfig, a.logger, a.tracer, msgProcessor, serviceRegistry)
	g.Go(func() error {
		a.logger.Info("Starting gRPC server...")
		return rpcServer.Start()
	})
	g.Go(func() error {
		<-ctx.Done()
		rpcServer.Stop()
		return nil
	})
}

func (a *App) startMessageProcessor(ctx context.Context, messenger messaging.Messenger, serviceRegistry *messaging.ServiceRegistry, responseHandler messaging.ResponseHandler, g *errgroup.Group, userIDUpdated chan string) messaging.Processor {
	msgProcessor := messaging.NewProcessor(messenger, a.logger, a.cfg.ProcessorConfig, serviceRegistry, responseHandler)
	g.Go(func() error {
		// Wait for userID to be passed
		userID := <-userIDUpdated
		close(userIDUpdated)
		msgProcessor.SetUserID(userID)
		a.logger.Info("Starting message processor...")
		msgProcessor.Start(ctx)
		return nil
	})
	return msgProcessor
}

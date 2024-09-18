package app

import (
	"context"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/events"
	"github.com/chain4travel/camino-messenger-bot/internal/evm"
	"github.com/chain4travel/camino-messenger-bot/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/utils/constants"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/common"
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
	defer logger.Sync() //nolint:errcheck

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	// TODO do proper DI with FX

	serviceRegistry := messaging.NewServiceRegistry(a.logger)
	// start rpc client if host is provided, otherwise bot serves as a distributor bot (rpc server)
	if a.cfg.PartnerPluginConfig.Host != "" {
		a.startRPCClient(gCtx, g, serviceRegistry)
	} else {
		a.logger.Infof("No host for partner plugin provided, bot will serve as a distributor bot.")
		serviceRegistry.RegisterServices(a.cfg.SupportedRequestTypes, nil)
	}
	// start messenger (receiver)
	messenger, userIDUpdatedChan := a.startMessenger(gCtx, g)

	evmClient, err := evm.NewClient(a.cfg.EvmConfig)
	if err != nil {
		a.logger.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	//
	// FIXME: REMOVE AFTER DEVELOPMENT
	//

	// get booking token address
	bookingTokenAddr := common.HexToAddress(a.cfg.BookingTokenAddress)

	// create and start event listener
	eventListener := events.NewEventListener(evmClient, a.logger)
	a.logger.Info("Event listener created")

	// Register an event handler for the BookingToken "TokenBought" event
	_, err = eventListener.RegisterTokenReservedHandler(bookingTokenAddr, nil, nil, nil, func(event interface{}) {
		e := event.(*bookingtoken.BookingtokenTokenReserved)
		a.logger.Infof("TokenReserved event: TokenId %s, Reserved For %s by Supplier %s", e.TokenId.String(), e.ReservedFor.Hex(), e.Supplier.Hex())
	})

	if err != nil {
		a.logger.Fatalf("Failed to register TokenReserved handler: %v", err)
	}

	cmAccountAddr := common.HexToAddress("0xd41786599F2B225A5A1eA35cDc4A2a6Fa9E92BeA")

	// Test filtering
	serviceName := []string{"cmp.services.ping.v1.PingService"}

	// Register an event handler for CMAccount's ServiceAdded event only for service name: "cmp.services.ping.v1.PingService"
	_, err = eventListener.RegisterServiceAddedHandler(cmAccountAddr, serviceName, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceAdded)
		a.logger.Infof("ServiceAdded event: ServiceHash %s", e.ServiceName)
	})

	if err != nil {
		a.logger.Fatalf("Failed to register ServiceAdded handler: %v", err)
	}

	// Register an event handler for CMAccount's ServiceRemoved event for any service name
	removeHandle, err := eventListener.RegisterServiceRemovedHandler(cmAccountAddr, nil, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceRemoved)
		a.logger.Infof("ServiceRemoved event: ServiceHash %s", e.ServiceName)
	})

	// Register an event handler for CMAccount's ServiceRemoved event for any service name
	_, err = eventListener.RegisterServiceRemovedHandler(cmAccountAddr, nil, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceRemoved)
		a.logger.Infof("#2 ServiceRemoved event: ServiceHash %s", e.ServiceName)
		// Unsubscribe #1 when we receive a remove event
		removeHandle.Unsubscribe()
	})

	if err != nil {
		a.logger.Fatalf("Failed to register ServiceRemoved handler: %v", err)
	}

	//
	// FIXME END: REMOVE AFTER DEVELOPMENT
	//

	// create response handler
	responseHandler, err := messaging.NewResponseHandler(evmClient, a.logger, &a.cfg.EvmConfig)
	if err != nil {
		a.logger.Fatalf("Failed to create to evm client: %v", err)
	}

	identificationHandler, err := messaging.NewIdentificationHandler(evmClient, a.logger, &a.cfg.EvmConfig, &a.cfg.MatrixConfig)
	if err != nil {
		a.logger.Fatalf("Failed to create to evm client: %v", err)
	}

	// start msg processor
	msgProcessor := a.startMessageProcessor(ctx, messenger, serviceRegistry, responseHandler, identificationHandler, g, userIDUpdatedChan)

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

func (a *App) startRPCServer(ctx context.Context, msgProcessor messaging.Processor, serviceRegistry messaging.ServiceRegistry, g *errgroup.Group) {
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

func (a *App) startMessageProcessor(ctx context.Context, messenger messaging.Messenger, serviceRegistry messaging.ServiceRegistry, responseHandler messaging.ResponseHandler, identificationHandler messaging.IdentificationHandler, g *errgroup.Group, userIDUpdated chan string) messaging.Processor {
	msgProcessor := messaging.NewProcessor(messenger, a.logger, a.cfg.ProcessorConfig, serviceRegistry, responseHandler, identificationHandler)
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

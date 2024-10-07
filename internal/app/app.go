package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/internal/scheduler"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"maunium.net/go/mautrix/id"
)

// TODO @nikos do proper DI with FX

const (
	cashInJobName = "cash_in"
	appName       = "camino-messenger-bot"
)

func NewApp(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) (*App, error) {
	// c-chain evm client && chain id
	evmClient, err := ethclient.Dial(cfg.CChainRPCURL.String())
	if err != nil {
		logger.Errorf("Failed to connect to the Ethereum client: %v", err)
		return nil, err
	}

	chainID, err := evmClient.NetworkID(ctx)
	if err != nil {
		logger.Errorf("Failed to fetch chain id: %v", err)
		return nil, err
	}

	storage, err := storage.New(ctx, logger, cfg.DBConfig)
	if err != nil {
		logger.Errorf("Failed to create storage: %v", err)
		return nil, err
	}

	// partner-plugin rpc client
	var rpcClient *client.RPCClient
	if cfg.PartnerPluginConfig.HostURL != nil {
		rpcClient, err = client.NewClient(cfg.PartnerPluginConfig, logger)
		if err != nil {
			logger.Errorf("Failed to create rpc client: %v", err)
			return nil, err
		}
	}

	// register supported services, check if they actually supported by bot
	serviceRegistry, hasSupportedServices, err := messaging.NewServiceRegistry(
		cfg.CMAccountAddress,
		evmClient,
		logger,
		rpcClient,
	)
	if err != nil {
		logger.Errorf("Failed to create service registry: %v", err)
		return nil, err
	}

	if !hasSupportedServices && rpcClient != nil {
		logger.Warn("Bot doesn't support any services, but has partner plugin rpc client configured")
	}

	// messaging components
	responseHandler, err := messaging.NewResponseHandler(
		cfg.BotKey,
		evmClient,
		logger,
		cfg.CMAccountAddress,
		cfg.BookingTokenAddress,
		serviceRegistry,
	)
	if err != nil {
		logger.Errorf("Failed to create response handler: %v", err)
		return nil, err
	}

	identificationHandler, err := messaging.NewIdentificationHandler(
		evmClient,
		logger,
		cfg.CMAccountAddress,
		cfg.MatrixConfig.HostURL,
	)
	if err != nil {
		logger.Errorf("Failed to create identification handler: %v", err)
		return nil, err
	}

	chequeHandler, err := messaging.NewChequeHandler(
		logger,
		evmClient,
		cfg.BotKey,
		cfg.CMAccountAddress,
		chainID,
		storage,
		serviceRegistry,
		cfg.MinChequeDurationUntilExpiration,
		cfg.ChequeExpirationTime,
	)
	if err != nil {
		logger.Errorf("Failed to create cheque handler: %v", err)
		return nil, err
	}

	matrixMessenger := matrix.NewMessenger(cfg.MatrixConfig, cfg.BotKey, logger)

	messageProcessor := messaging.NewProcessor(
		matrixMessenger,
		logger,
		cfg.ResponseTimeout,
		cfg.CMAccountAddress,
		cfg.NetworkFeeRecipientBotAddress,
		cfg.NetworkFeeRecipientCMAccountAddress,
		serviceRegistry,
		responseHandler,
		identificationHandler,
		chequeHandler,
		messaging.NewCompressor(compression.MaxChunkSize),
	)

	// rpc server for incoming requests
	// TODO@ disable if we don't have port provided, e.g. its supplier bot?
	rpcServer := server.NewServer(
		cfg.RPCServerConfig,
		logger,
		tracer,
		messageProcessor,
		serviceRegistry,
	)

	// scheduler for periodic tasks (e.g. cheques cash-in)
	scheduler := scheduler.New(ctx, logger, storage)
	scheduler.RegisterJobHandler(cashInJobName, func() {
		_ = chequeHandler.CashIn(context.Background())
	})

	return &App{
		cfg:              cfg,
		logger:           logger,
		scheduler:        scheduler,
		chequeHandler:    chequeHandler,
		rpcClient:        rpcClient,
		rpcServer:        rpcServer,
		messageProcessor: messageProcessor,
		messenger:        matrixMessenger,
	}, nil
}

type App struct {
	cfg              *config.Config
	logger           *zap.SugaredLogger
	tracer           tracing.Tracer
	scheduler        scheduler.Scheduler
	chequeHandler    messaging.ChequeHandler
	rpcClient        *client.RPCClient
	rpcServer        server.Server
	messageProcessor messaging.Processor
	messenger        messaging.Messenger
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-ctx.Done()
		return a.rpcClient.Shutdown()
	})

	userIDUpdatedChan := make(chan id.UserID) // Channel to pass the userID
	g.Go(func() error {
		a.logger.Info("Starting message receiver...")
		userID, err := a.messenger.StartReceiver()
		if err != nil {
			panic(err)
		}
		userIDUpdatedChan <- userID // Pass userID through the channel
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		return a.messenger.StopReceiver()
	})

	if err := a.chequeHandler.CheckCashInStatus(ctx); err != nil {
		a.logger.Errorf("start-up cash-in status check failed: %v", err)
	}

	g.Go(func() error {
		// Wait for userID to be passed
		userID := <-userIDUpdatedChan
		close(userIDUpdatedChan)
		a.messageProcessor.SetUserID(userID)
		a.logger.Info("Starting message processor...")
		a.messageProcessor.Start(ctx)
		return nil
	})

	if err := a.scheduler.Schedule(ctx, time.Duration(a.cfg.CashInPeriod)*time.Second, cashInJobName); err != nil { //nolint:gosec
		a.logger.Fatalf("Failed to schedule cash in job: %v", err)
	}

	if err := a.scheduler.Start(ctx); err != nil {
		a.logger.Fatalf("Failed to start scheduler: %v", err)
	}

	// init tracer
	tracer := a.initTracer()
	defer func() {
		if err := tracer.Shutdown(); err != nil {
			a.logger.Fatal("failed to shutdown tracer: %w", err)
		}
	}()

	// start rpc server
	g.Go(func() error {
		a.logger.Info("Starting gRPC server...")
		return a.rpcServer.Start()
	})
	g.Go(func() error {
		<-ctx.Done()
		a.rpcServer.Stop()
		return nil
	})

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
		tracer, err = tracing.NewTracer(&a.cfg.TracingConfig, fmt.Sprintf("%s:%d", appName, a.cfg.RPCServerConfig.Port))
	} else {
		tracer, err = tracing.NewNoOpTracer()
	}
	if err != nil {
		a.logger.Fatal("failed to initialize tracer: %w", err)
	}
	a.tracer = tracer
	return tracer
}

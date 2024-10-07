package app

import (
	"context"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"maunium.net/go/mautrix/id"

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

	// tracer
	var tracer tracing.Tracer
	if cfg.TracingConfig.Enabled {
		tracer, err = tracing.NewTracer(
			cfg.TracingConfig,
			fmt.Sprintf("%s:%d", appName, cfg.RPCServerConfig.Port),
		)
	} else {
		tracer, err = tracing.NewNoOpTracer()
	}
	if err != nil {
		logger.Errorf("Failed to initialize tracer: %v", err)
		return nil, err
	}

	// database
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

	botAddress := crypto.PubkeyToAddress(cfg.BotKey.PublicKey)
	botUserID := messaging.UserIDFromAddress(botAddress, cfg.MatrixConfig.HostURL.String())

	messageProcessor := messaging.NewProcessor(
		matrixMessenger,
		logger,
		cfg.ResponseTimeout,
		botUserID,
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
	rpcServer, err := server.NewServer(
		cfg.RPCServerConfig,
		logger,
		tracer,
		messageProcessor,
		serviceRegistry,
	)
	if err != nil {
		logger.Errorf("Failed to create rpc server: %v", err)
		return nil, err
	}

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
		tracer:           tracer,
		botUserID:        botUserID,
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
	botUserID        id.UserID
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.tracer.Shutdown(); err != nil {
			a.logger.Errorf("failed to shutdown tracer: %v", err)
		}
	}()

	g, gCtx := errgroup.WithContext(ctx) // error here will call gCtx.cancel() and finish other Go-s

	// run

	g.Go(func() error {
		a.logger.Info("Starting gRPC server...")
		return a.rpcServer.Start()
	})

	g.Go(func() error {
		a.logger.Info("Starting start-up cash-in status check...")
		if err := a.chequeHandler.CheckCashInStatus(gCtx); err != nil {
			return fmt.Errorf("failed to check start-up cash-in status: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		a.logger.Info("Starting scheduler...")
		if err := a.scheduler.Schedule(gCtx, a.cfg.CashInPeriod, cashInJobName); err != nil {
			return fmt.Errorf("failed to schedule cash in job: %w", err)
		}
		if err := a.scheduler.Start(gCtx); err != nil {
			return fmt.Errorf("failed to start scheduler: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		a.logger.Info("Starting message receiver...")
		matrixUserID, err := a.messenger.StartReceiver()
		if err != nil {
			return fmt.Errorf("failed to start message receiver: %w", err)
		}
		if a.botUserID != matrixUserID {
			return fmt.Errorf("bot user ID mismatch: expected %s, got %s", a.botUserID, matrixUserID)
		}
		return nil
	})

	g.Go(func() error {
		a.logger.Info("Starting message processor...")
		a.messageProcessor.Start(gCtx)
		return nil
	})

	// stop
	// <-gCtx.Done() means that all "run" goroutines are finished

	g.Go(func() error {
		<-ctx.Done()
		return a.rpcClient.Shutdown()
	})

	g.Go(func() error {
		<-ctx.Done()
		a.rpcServer.Stop()
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		return a.messenger.StopReceiver()
	})

	// wait

	err := g.Wait()
	if err != nil {
		a.logger.Error(err) // will log first run/stop error
	}
	return err
}

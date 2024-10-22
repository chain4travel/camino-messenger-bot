package app

import (
	"context"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jonboulle/clockwork"
	"maunium.net/go/mautrix/id"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
	chequeHandlerStorage "github.com/chain4travel/camino-messenger-bot/pkg/chequehandler/storage/sqlite"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	"github.com/chain4travel/camino-messenger-bot/pkg/scheduler"
	scheduler_storage "github.com/chain4travel/camino-messenger-bot/pkg/scheduler/storage/sqlite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	cashInJobName        = "cash_in"
	appName              = "camino-messenger-bot"
	cmAccountsCacheSize  = 100
	erc20CacheSize       = 100
	cashInTxIssueTimeout = 10 * time.Second
)

func NewApp(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) (*App, error) {
	// c-chain evm client && chain id
	evmClient, err := ethclient.Dial(cfg.ChainRPCURL.String())
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
	if cfg.Tracing.Enabled {
		tracer, err = tracing.NewTracer(
			cfg.Tracing,
			fmt.Sprintf("%s:%d", appName, cfg.RPCServer.Port),
		)
	} else {
		tracer, err = tracing.NewNoOpTracer()
	}
	if err != nil {
		logger.Errorf("Failed to initialize tracer: %v", err)
		return nil, err
	}

	// partner-plugin rpc client
	rpcClient, err := client.NewClient(cfg.PartnerPlugin, logger)
	if err != nil {
		logger.Errorf("Failed to create rpc client: %v", err)
		return nil, err
	}

	// register supported services, check if they actually supported by bot
	serviceRegistry, err := messaging.NewServiceRegistry(
		cfg.CMAccountAddress,
		evmClient,
		logger,
		rpcClient,
	)
	if err != nil {
		logger.Errorf("Failed to create service registry: %v", err)
		return nil, err
	}

	// messaging components
	cmAccounts, err := cmaccounts.NewService(
		logger,
		cmAccountsCacheSize,
		evmClient,
	)
	if err != nil {
		logger.Errorf("Failed to create cm accounts service: %v", err)
		return nil, err
	}

	responseHandler, err := messaging.NewResponseHandler(
		cfg.BotKey,
		evmClient,
		logger,
		cfg.CMAccountAddress,
		cfg.BookingTokenAddress,
		serviceRegistry,
		cmAccounts,
		erc20CacheSize,
	)
	if err != nil {
		logger.Errorf("Failed to create response handler: %v", err)
		return nil, err
	}

	chequeHandlerStorage, err := chequeHandlerStorage.New(
		ctx,
		logger,
		sqlite.DBConfig(cfg.DB.ChequeHandler),
	)
	if err != nil {
		logger.Errorf("Failed to create cheque handler storage: %v", err)
		return nil, err
	}

	chequeHandler, err := chequehandler.NewChequeHandler(
		logger,
		evmClient,
		cfg.BotKey,
		cfg.CMAccountAddress,
		chainID,
		chequeHandlerStorage,
		cmAccounts,
		cfg.MinChequeDurationUntilExpiration,
		cfg.ChequeExpirationTime,
		cashInTxIssueTimeout,
	)
	if err != nil {
		logger.Errorf("Failed to create cheque handler: %v", err)
		return nil, err
	}

	matrixMessenger, err := matrix.NewMessenger(cfg.Matrix, cfg.BotKey, logger)
	if err != nil {
		logger.Errorf("Failed to create matrix messenger: %v", err)
		return nil, err
	}

	botAddress := crypto.PubkeyToAddress(cfg.BotKey.PublicKey)
	botUserID := messaging.UserIDFromAddress(botAddress, cfg.Matrix.HostURL.String())

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
		chequeHandler,
		messaging.NewCompressor(compression.MaxChunkSize),
		cmAccounts,
	)

	// rpc server for incoming requests
	rpcServer, err := server.NewServer(
		cfg.RPCServer,
		logger,
		tracer,
		messageProcessor,
		serviceRegistry,
		cfg.DeveloperMode,
	)
	if err != nil {
		logger.Errorf("Failed to create rpc server: %v", err)
		return nil, err
	}

	// scheduler for periodic tasks (e.g. cheques cash-in)

	storage, err := scheduler_storage.New(ctx, logger, sqlite.DBConfig(cfg.DB.Scheduler))
	if err != nil {
		logger.Errorf("Failed to create storage: %v", err)
		return nil, err
	}

	scheduler := scheduler.New(logger, storage, clockwork.NewRealClock())
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
	chequeHandler    chequehandler.ChequeHandler
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

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		g.Go(func() error {
			a.logger.Info("Starting gRPC server...")
			return a.rpcServer.Start()
		})
	}

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

	if a.rpcClient != nil { // rpcClient will be nil, if its disabled in partner plugin config section
		g.Go(func() error {
			<-ctx.Done()
			return a.rpcClient.Shutdown()
		})
	}

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		g.Go(func() error {
			<-ctx.Done()
			a.rpcServer.Stop()
			return nil
		})
	}

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

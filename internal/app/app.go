// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package app

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jonboulle/clockwork"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"maunium.net/go/mautrix/id"

	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/cancellation"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	eventlistener "github.com/chain4travel/camino-messenger-bot/v11/internal/event_listener"
	eventlistener_storage "github.com/chain4travel/camino-messenger-bot/v11/internal/event_listener/storage/sqlite"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/chequehandler"
	chequeHandlerStorage "github.com/chain4travel/camino-messenger-bot/v11/pkg/chequehandler/storage/sqlite"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/erc20"
	matrixPkg "github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/scheduler"
	scheduler_storage "github.com/chain4travel/camino-messenger-bot/v11/pkg/scheduler/storage/sqlite"
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
	evmClient, err := ethclient.Dial(cfg.ChainRPCURL)
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

	// partner plugin to handle partner-plugin related logic and communication
	var partnerPlugin partnerplugin.PartnerPlugin
	if cfg.PartnerPlugin.Enabled {
		partnerPlugin = partnerplugin.New(
			logger,
			tracer,
			rpcClient,
			cfg.ResponseTimeout,
		)
	}

	// blockchain services

	cmAccounts, err := cmaccounts.NewService(
		logger,
		cmAccountsCacheSize,
		evmClient,
	)
	if err != nil {
		logger.Errorf("Failed to create cm accounts service: %v", err)
		return nil, err
	}

	// TODO: @VjeraTurk Ensure multiple versions compatibility
	cmAccountUpToDate, err := cmAccounts.IsCMAccountImplementationUpToDate(ctx, cfg.CMAccountAddress)
	if err != nil {
		logger.Errorf("Failed to compare implementations: %v", err)
		return nil, err
	}

	if !cmAccountUpToDate {
		logger.Warn("⏫ CMAccount needs an upgrade!")
	} else {
		logger.Info("✅ CMAccount is using the latest implementation.")
	}

	erc20, err := erc20.NewERC20Service(evmClient, erc20CacheSize)
	if err != nil {
		return nil, err
	}

	bookingService, err := booking.NewService(
		evmClient,
		cfg.BookingTokenAddress,
		cfg.CMAccountAddress,
		cfg.BotKey,
		chainID,
		logger,
		cmAccounts,
	)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	priceHandler := common.NewPriceHandler(erc20)

	// event listener with additional logic for subscribing and reacting on blockchain events

	eventListenerStorage, err := eventlistener_storage.New(ctx, logger, cfg.DB.EventListener.DBPath)
	if err != nil {
		logger.Errorf("Failed to create event listener storage: %v", err)
		return nil, err
	}

	eventListener, err := eventlistener.New(
		ctx,
		logger,
		eventListenerStorage,
		evmClient,
		cfg.BookingTokenAddress,
		partnerPlugin,
		bookingService,
		cmAccounts,
		cfg.RecordExpiration,
	)
	if err != nil {
		logger.Errorf("Failed to create event listener: %v", err)
		return nil, err
	}

	// messaging components

	responseHeaderHandler := common.NewResponseHeaderHandler(logger)

	responseHandler, err := messaging.NewResponseHandler(
		logger,
		cfg.CMAccountAddress,
		eventListener,
		bookingService,
		responseHeaderHandler,
		priceHandler,
		cfg.E2ETestMode,
	)
	if err != nil {
		logger.Errorf("Failed to create response handler: %v", err)
		return nil, err
	}

	chequeHandlerStorage, err := chequeHandlerStorage.New(
		ctx,
		logger,
		cfg.DB.ChequeHandler.DBPath,
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

	// get matrix hostname without schema
	matrixHostname := cfg.Matrix.Host
	if !strings.Contains(matrixHostname, "://") {
		// Add dummy protocol to make the url pkg happy,
		// we just want to extract the hostname
		matrixHostname = "dummy://" + matrixHostname
	}
	u, err := url.Parse(matrixHostname)
	if err != nil {
		return nil, fmt.Errorf("failed to parse matrix host: %w", err)
	}
	matrixHostname = u.Hostname()

	botAddress := crypto.PubkeyToAddress(cfg.BotKey.PublicKey)
	botUserID := matrixPkg.UserIDFromAddress(botAddress, matrixHostname)

	messageProcessor := messaging.NewMessageProcessor(
		matrixMessenger,
		logger,
		cfg.ResponseTimeout,
		botUserID,
		cfg.CMAccountAddress,
		cfg.NetworkFeeRecipientBotAddress,
		cfg.NetworkFeeRecipientCMAccountAddress,
		serviceRegistry,
		responseHandler,
		partnerPlugin,
		chequeHandler,
		messaging.NewCompressor(compression.MaxChunkSize),
		cmAccounts,
		responseHeaderHandler,
	)

	cancellationService := cancellation.NewService(
		logger,
		cfg.BotKey,
		cfg.CMAccountAddress,
		cmAccounts,
		priceHandler,
	)

	// rpc server for incoming requests
	rpcServer, err := server.NewServer(
		cfg.RPCServer,
		logger,
		responseHeaderHandler,
		tracer,
		messageProcessor,
		serviceRegistry,
		cancellationService,
		cfg.DeveloperMode,
	)
	if err != nil {
		logger.Errorf("Failed to create rpc server: %v", err)
		return nil, err
	}

	// scheduler for periodic tasks (e.g. cheques cash-in)

	storage, err := scheduler_storage.New(ctx, logger, cfg.DB.Scheduler.DBPath)
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
		eventListener:    eventListener,
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
	eventListener    eventlistener.EventListener
	rpcClient        *client.RPCClient
	rpcServer        server.Server
	messageProcessor messaging.MessageProcessor
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

	messengerReceiverStarted := make(chan struct{})
	cashInStatusCheckDone := make(chan struct{})
	schedulerStarted := make(chan struct{})
	messageProcessorStarted := make(chan struct{})

	g.Go(func() error {
		a.logger.Info("Starting start-up cash-in status check...")
		if err := a.chequeHandler.CheckCashInStatus(gCtx); err != nil {
			return fmt.Errorf("failed to check start-up cash-in status: %w", err)
		}
		a.logger.Info("Start-up cash-in status check done.")
		close(cashInStatusCheckDone)
		return nil
	})

	eventListenerStarted := make(chan struct{})
	g.Go(func() error {
		a.logger.Info("Starting event listener...")
		if err := a.eventListener.Start(gCtx); err != nil {
			return fmt.Errorf("failed to start event listener: %w", err)
		}
		a.logger.Info("Event listener started.")
		close(eventListenerStarted)
		return nil
	})

	g.Go(func() error {
		if !awaitChan(gCtx, cashInStatusCheckDone) {
			return nil
		}

		a.logger.Info("Starting scheduler...")

		if err := a.scheduler.Schedule(gCtx, a.cfg.CashInPeriod, cashInJobName); err != nil {
			return fmt.Errorf("failed to schedule cash in job: %w", err)
		}
		if err := a.scheduler.Start(gCtx); err != nil {
			return fmt.Errorf("failed to start scheduler: %w", err)
		}
		a.logger.Info("Scheduler started.")
		close(schedulerStarted)
		return nil
	})

	g.Go(func() error {
		a.logger.Info("Starting message processor...")
		close(messageProcessorStarted)
		a.messageProcessor.Start(gCtx)
		return nil
	})

	g.Go(func() error {
		if !awaitChans(gCtx,
			cashInStatusCheckDone,
			schedulerStarted,
			messageProcessorStarted,
			eventListenerStarted,
		) {
			return nil
		}
		a.logger.Info("Starting message receiver...")
		matrixUserID, err := a.messenger.StartReceiver()
		if err != nil {
			return fmt.Errorf("failed to start message receiver: %w", err)
		}
		if a.botUserID != matrixUserID {
			return fmt.Errorf("bot user ID mismatch: expected %s, got %s", a.botUserID, matrixUserID)
		}
		close(messengerReceiverStarted)
		return nil
	})

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		g.Go(func() error {
			if !awaitChan(gCtx, messengerReceiverStarted) {
				return nil
			}

			a.logger.Info("Starting gRPC server...")
			return a.rpcServer.Start()
		})
	}

	// stop

	if a.rpcClient != nil { // rpcClient will be nil, if its disabled in partner plugin config section
		g.Go(func() error {
			<-gCtx.Done()
			a.logger.Info("Stopping gRPC client...")
			return a.rpcClient.Shutdown()
		})
	}

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		g.Go(func() error {
			<-gCtx.Done()
			a.logger.Info("Stopping gRPC server...")
			a.rpcServer.Stop()
			return nil
		})
	}

	g.Go(func() error {
		<-gCtx.Done()
		a.logger.Info("Stopping message receiver...")
		return a.messenger.StopReceiver()
	})

	g.Go(func() error {
		<-gCtx.Done()
		a.logger.Info("Stopping scheduler...")
		return a.scheduler.Stop()
	})

	g.Go(func() error {
		<-gCtx.Done()
		a.logger.Info("Stopping event listener...")
		a.eventListener.Stop()
		return nil
	})

	// wait

	err := g.Wait()
	if err != nil {
		a.logger.Error(err) // will log first run/stop error
	}

	return err
}

func awaitChan(ctx context.Context, ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

func awaitChans(ctx context.Context, chans ...<-chan struct{}) bool {
	for _, ch := range chans {
		if !awaitChan(ctx, ch) {
			return false
		}
	}
	return true
}

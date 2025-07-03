// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package app

import (
	"context"
	"errors"
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
	"google.golang.org/grpc"
	"maunium.net/go/mautrix/id"

	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/cancellation"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/eventlistener"
	eventlistener_storage "github.com/chain4travel/camino-messenger-bot/v11/internal/eventlistener/storage/sqlite"
	matrix_client "github.com/chain4travel/camino-messenger-bot/v11/internal/matrix/client"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/matrix/messenger"
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
			ctx,
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
	} else {
		logger.Info("Partner plugin is disabled in config, skipping partner plugin (gRPC client) initialization.")
	}

	// blockchain services

	cmAccounts, err := cmaccounts.NewService(
		ctx,
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

	matrixClient, err := matrix_client.New(
		ctx,
		logger,
		cfg.Matrix.Host,
		cfg.Matrix.Store,
		cfg.BotKey,
		botUserID,
	)
	if err != nil {
		logger.Errorf("failed to create matrix client: %v", err)
		return nil, err
	}

	matrixMessenger, err := messenger.NewMessenger(
		logger,
		matrixClient,
		&compression.ZSTDDecompressor{},
		cfg.BotKey,
		botUserID,
	)
	if err != nil {
		logger.Errorf("Failed to create matrix messenger: %v", err)
		return nil, err
	}

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
		messaging.NewCompressor(matrix_client.MaxChunkSize),
		cmAccounts,
		responseHeaderHandler,
		cfg.MaxAllowedServiceFee,
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
		// we use background context, because we want to try to shutdown tracer gracefully regardless
		if err := a.tracer.Shutdown(context.Background()); err != nil {
			a.logger.Errorf("failed to shutdown tracer: %v", err)
		}
	}()

	g, ctx := errgroup.WithContext(ctx) // error here will call ctx.cancel() and finish other Go-s

	// run

	messengerStarted := make(chan struct{})
	cashInStatusCheckDone := make(chan struct{})
	schedulerStarted := make(chan struct{})
	messageProcessorStarted := make(chan struct{})

	a.safeGo(g, func() error {
		a.logger.Info("Starting start-up cash-in status check...")
		if err := a.chequeHandler.CheckCashInStatus(ctx); err != nil {
			return fmt.Errorf("failed to do start-up cash-in status check: %w", err)
		}
		a.logger.Info("Start-up cash-in status check done.")
		close(cashInStatusCheckDone)
		return nil
	})

	eventListenerStarted := make(chan struct{})
	a.safeGo(g, func() error {
		a.logger.Info("Starting event listener...")
		errChan, err := a.eventListener.Start(ctx)
		if err != nil {
			return fmt.Errorf("failed to start event listener: %w", err)
		}
		a.logger.Info("Event listener started.")
		close(eventListenerStarted)

		if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Errorf("Event listener failed with error: %v", err)
			return err
		}
		return nil
	})

	a.safeGo(g, func() error {
		if !awaitChan(ctx, cashInStatusCheckDone) {
			return nil
		}
		a.logger.Info("Starting scheduler...")

		a.scheduler.RegisterJobHandler(cashInJobName, func() {
			if err := a.chequeHandler.CashIn(ctx); err != nil && !errors.Is(err, context.Canceled) {
				a.logger.Errorf("Failed to do scheduled cash in: %v", err)
				return
			}
		})

		if err := a.scheduler.Schedule(ctx, a.cfg.CashInPeriod, cashInJobName); err != nil {
			return fmt.Errorf("failed to schedule cash in job: %w", err)
		}
		if err := a.scheduler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start scheduler: %w", err)
		}

		a.logger.Info("Scheduler started.")
		close(schedulerStarted)

		return nil
	})

	a.safeGo(g, func() error {
		a.logger.Info("Starting message processor...")
		a.messageProcessor.Start(ctx)
		a.logger.Info("Message processor started.")
		close(messageProcessorStarted)
		return nil
	})

	a.safeGo(g, func() error {
		if !awaitChans(ctx,
			cashInStatusCheckDone,
			schedulerStarted,
			messageProcessorStarted,
			eventListenerStarted,
		) {
			return nil
		}

		a.logger.Info("Starting matrix messenger...")

		errChan, err := a.messenger.Start(ctx)
		if err != nil {
			return fmt.Errorf("failed to start matrix messenger: %w", err)
		}

		a.logger.Info("Matrix messenger started.")
		close(messengerStarted)

		if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Errorf("Matrix messenger exited with error: %v", err)
			return err
		}
		return nil
	})

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		a.safeGo(g, func() error {
			if !awaitChan(ctx, messengerStarted) {
				return nil
			}

			a.logger.Info("Starting gRPC server...")
			errChan, err := a.rpcServer.Start()
			if err != nil {
				return fmt.Errorf("failed to start gRPC server: %w", err)
			}

			a.logger.Info("gRPC server started.")

			if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, grpc.ErrServerStopped) {
				a.logger.Errorf("gRPC server stopped with error: %v", err)
				return err
			}
			return nil
		})
	} else {
		a.logger.Info("gRPC server is disabled in config, skipping gRPC server start.")
	}

	// stop

	if a.rpcClient != nil { // rpcClient will be nil, if its disabled in partner plugin config section
		a.safeGo(g, func() error {
			<-ctx.Done()
			a.logger.Info("Stopping gRPC client...")
			if err := a.rpcClient.Shutdown(); err != nil && !errors.Is(err, context.Canceled) {
				a.logger.Errorf("Failed to stop gRPC client: %v", err)
				return err
			}
			a.logger.Info("gRPC client stopped.")
			return nil
		})
	}

	if a.rpcServer != nil { // rpcServer will be nil, if its disabled in config
		a.safeGo(g, func() error {
			<-ctx.Done()
			a.logger.Info("Stopping gRPC server...")
			a.rpcServer.Stop()
			a.logger.Info("gRPC server stopped.")
			return nil
		})
	}

	a.safeGo(g, func() error {
		<-ctx.Done()
		a.logger.Info("Stopping matrix messenger...")
		if err := a.messenger.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Errorf("Failed to stop matrix messenger: %v", err)
			return err
		}
		a.logger.Info("Matrix messenger stopped.")
		return nil
	})

	a.safeGo(g, func() error {
		<-ctx.Done()
		a.logger.Info("Stopping scheduler...")
		a.scheduler.Stop()
		a.logger.Info("Scheduler stopped.")
		return nil
	})

	a.safeGo(g, func() error {
		<-ctx.Done()
		a.logger.Info("Stopping event listener...")
		a.eventListener.Stop()
		a.logger.Info("Event listener stopped.")
		return nil
	})

	// Message processor is stopped by canceling context, so we don't need to explicitly stop it.

	// wait

	err := g.Wait()
	if err != nil {
		a.logger.Errorf("App stopped with error: %v", err) // will log first run/stop error
	}

	return err
}

func (a *App) safeGo(g *errgroup.Group, fn func() error) {
	g.Go(func() (err error) {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				err = fmt.Errorf("panic: %v", panicErr) // err will be returned
				a.logger.Errorf("recovered from panic: %v", err)
			}
		}()
		return fn()
	})
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

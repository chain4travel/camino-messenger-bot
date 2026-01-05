// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package app

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/chain4travel/camino-messenger-bot/v12/config"
	cancellation_v1 "github.com/chain4travel/camino-messenger-bot/v12/internal/cancellation/v1"
	cancellation_v2 "github.com/chain4travel/camino-messenger-bot/v12/internal/cancellation/v2"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/eventlistener"
	eventlistener_storage "github.com/chain4travel/camino-messenger-bot/v12/internal/eventlistener/storage/sqlite"
	matrix_client "github.com/chain4travel/camino-messenger-bot/v12/internal/matrix/client"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/matrix/messenger"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encoding"
	messagesEncoderDecoderStorage "github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encoding/storage/sqlite"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/price"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/resolver"
	resolver_storage "github.com/chain4travel/camino-messenger-bot/v12/internal/resolver/storage/sqlite"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/chequehandler"
	chequeHandlerStorage "github.com/chain4travel/camino-messenger-bot/v12/pkg/chequehandler/storage/sqlite"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/erc20"
	matrixPkg "github.com/chain4travel/camino-messenger-bot/v12/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/scheduler"
	scheduler_storage "github.com/chain4travel/camino-messenger-bot/v12/pkg/scheduler/storage/sqlite"
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
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	chainID, err := evmClient.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chain id: %w", err)
	}

	// partner-plugin rpc client
	rpcClient, err := client.NewClient(cfg.PartnerPlugin, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client: %w", err)
	}

	// register supported services, check if they actually supported by bot
	serviceRegistry, err := messaging.NewServiceRegistry(
		cfg.CMAccountAddress,
		evmClient,
		logger,
		rpcClient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service registry: %w", err)
	}

	// partner plugin to handle partner-plugin related logic and communication
	var partnerPlugin partnerplugin.PartnerPlugin
	if cfg.PartnerPlugin.Enabled {
		partnerPlugin = partnerplugin.New(
			logger,
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
		return nil, fmt.Errorf("failed to create cm accounts service: %w", err)
	}

	// TODO: @VjeraTurk Ensure multiple versions compatibility
	cmAccountUpToDate, err := cmAccounts.IsCMAccountImplementationUpToDate(ctx, cfg.CMAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to compare CMAccount implementations: %w", err)
	}

	if !cmAccountUpToDate {
		logger.Warn("⏫ CMAccount needs an upgrade!")
	} else {
		logger.Info("✅ CMAccount is using the latest implementation.")
	}

	erc20, err := erc20.NewERC20Service(evmClient, erc20CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create erc20 service: %w", err)
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
		return nil, fmt.Errorf("failed to create booking service: %w", err)
	}

	priceHandler := price.NewPriceHandler(erc20)

	// event listener with additional logic for subscribing and reacting on blockchain events

	eventListenerStorage, err := eventlistener_storage.New(ctx, logger, cfg.DB.EventListener.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create event listener storage: %w", err)
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
		return nil, fmt.Errorf("failed to create event listener: %w", err)
	}

	// messaging components

	responseHandler, err := messaging.NewResponseHandler(
		logger,
		cfg.CMAccountAddress,
		eventListener,
		bookingService,
		priceHandler,
		cfg.E2ETestMode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create response handler: %w", err)
	}

	chequeHandlerStorage, err := chequeHandlerStorage.New(
		ctx,
		logger,
		cfg.DB.ChequeHandler.DBPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cheque handler storage: %w", err)
	}

	chequeHandler, err := chequehandler.NewChequeHandler(
		ctx,
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
		return nil, fmt.Errorf("failed to create cheque handler: %w", err)
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
		return nil, fmt.Errorf("failed to create matrix client: %w", err)
	}

	messagesEncoderDecoderStorage, err := messagesEncoderDecoderStorage.New(
		ctx,
		logger,
		cfg.DB.MessagesEncoderDecoder.DBPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages encoder/decoder storage: %w", err)
	}

	messagesEncoderDecoder, err := encoding.NewEncoderDecoder(
		logger,
		messagesEncoderDecoderStorage,
		matrix_client.MaxChunkSize,
		cfg.BotKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages encoder/decoder: %w", err)
	}

	resolverStorage, err := resolver_storage.New(
		ctx,
		logger,
		cfg.DB.Resolver.DBPath,
	)
	if err != nil {
		logger.Errorf("Failed to create resolver storage: %v", err)
		return nil, err
	}

	resolver := resolver.NewResolver(logger, cmAccounts, resolverStorage)

	matrixMessenger, err := messenger.NewMessenger(
		logger,
		matrixClient,
		cfg.BotKey,
		botUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create matrix messenger: %w", err)
	}

	messageProcessor := messaging.NewMessageProcessor(
		matrixMessenger,
		logger,
		cfg.ResponseTimeout,
		botAddress,
		cfg.CMAccountAddress,
		cfg.NetworkFeeRecipientBotAddress,
		cfg.NetworkFeeRecipientCMAccountAddress,
		serviceRegistry,
		responseHandler,
		partnerPlugin,
		chequeHandler,
		cmAccounts,
		cfg.MaxAllowedServiceFee,
		messagesEncoderDecoder,
		resolver,
	)

	cancellationV1Service := cancellation_v1.NewService(
		logger,
		cfg.BotKey,
		cfg.CMAccountAddress,
		cmAccounts,
		priceHandler,
	)

	cancellationV2Service := cancellation_v2.NewService(
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
		messageProcessor,
		serviceRegistry,
		cancellationV1Service,
		cancellationV2Service,
		cfg.DeveloperMode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc server: %w", err)
	}

	// scheduler for periodic tasks (e.g. cheques cash-in)

	storage, err := scheduler_storage.New(ctx, logger, cfg.DB.Scheduler.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler storage: %w", err)
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
		botUserID:        botUserID,
	}, nil
}

type App struct {
	cfg              *config.Config
	logger           *zap.SugaredLogger
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
			err = fmt.Errorf("event listener failed with error: %w", err)
			a.logger.Error(err)
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
			err = fmt.Errorf("matrix messenger failed with error: %w", err)
			a.logger.Error(err)
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
			errChan, err := a.rpcServer.Start(ctx)
			if err != nil {
				return fmt.Errorf("failed to start gRPC server: %w", err)
			}

			a.logger.Info("gRPC server started.")

			if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, grpc.ErrServerStopped) {
				err = fmt.Errorf("gRPC server failed with error: %w", err)
				a.logger.Error(err)
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
				err = fmt.Errorf("failed to stop gRPC client: %w", err)
				a.logger.Error(err)
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
			err = fmt.Errorf("failed to stop matrix messenger: %w", err)
			a.logger.Error(err)
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

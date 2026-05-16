// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package bot

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/chain4travel/camino-messenger-bot/v13/config"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/blockchain"
	e2eCommon "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/resources"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

const CashInPeriodSeconds = 3600 // 1h

func NewFactory(
	logger *zap.SugaredLogger,
	resourceManagerSession *resources.Session,
	e2eTmpDir string,
	binPath string,
	networkClient *blockchain.Client,
	matrix *matrix.ConduitServer,
	asb *matrix.AppService,
) *Factory {
	return &Factory{
		logger:                 logger,
		resourceManagerSession: resourceManagerSession,
		dir:                    path.Join(e2eTmpDir, "cmb"),
		binPath:                binPath,
		networkClient:          networkClient,
		matrix:                 matrix,
		asb:                    asb,
	}
}

type Factory struct {
	logger                 *zap.SugaredLogger
	resourceManagerSession *resources.Session
	dir                    string
	binPath                string
	networkClient          *blockchain.Client
	matrix                 *matrix.ConduitServer
	asb                    *matrix.AppService
	mutex                  sync.Mutex
	bots                   []*Bot
}

type options struct {
	skips               *Skip
	cashInPeriodSeconds int64
	services            []CMService
}

type Option func(*options)

func WithSkips(skips *Skip) Option {
	return func(o *options) { o.skips = skips }
}

// Intentionally skip some steps in bot creation.
// Only used for very specific testing purposes.
type Skip struct {
	// Skips the creation of the cm-account when setting up the bot.
	CMAccountCreation bool
	// Skips the transfer of funds to the cm-account owner.
	// Requires CMAccountCreation to be false.
	PrefundOwner bool
	// Skips the transfer of funds to the bot.
	// Requires CMAccountCreation to be false.
	PrefundBot bool
	// Skips the registration of the bot in the cm-account.
	// Requires CMAccountCreation and PrefundOwner to be false.
	BotRegistration bool
	// Skips the registration of services in the cm-account.
	// Requires CMAccountCreation and PrefundOwner to be false.
	ServiceRegistration bool
}

func WithCashInPeriod(cashInPeriodSeconds int64) Option {
	return func(o *options) { o.cashInPeriodSeconds = cashInPeriodSeconds }
}

func WithServices(services []CMService) Option {
	return func(o *options) { o.services = services }
}

type CMService struct {
	Name string
	Fee  int64
}

func (f *Factory) CreateBot(
	ctx context.Context,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	opts ...Option,
) (*Bot, error) {
	options := &options{
		skips:               &Skip{},
		cashInPeriodSeconds: CashInPeriodSeconds, // 1h
	}
	for _, opt := range opts {
		opt(options)
	}

	cmAccountOwnerKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	botKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	var cmAccountAddress common.Address
	if !options.skips.CMAccountCreation {
		botAddr := crypto.PubkeyToAddress(botKey.PublicKey)
		cmAccountOwnerAddress := crypto.PubkeyToAddress(cmAccountOwnerKey.PublicKey)
		cmAccountAddress, _, err = f.networkClient.CreateCMAccount(ctx, cmAccountOwnerKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create CM account: %w", err)
		}

		if !options.skips.PrefundOwner {
			if err := f.networkClient.Transfer(ctx, f.networkClient.PrefundedKeys()[0], cmAccountOwnerAddress, e2eCommon.DefaultCMAccountOwnerFunds); err != nil {
				return nil, fmt.Errorf("failed to transfer funds to cm account owner: %w", err)
			}

			if !options.skips.BotRegistration {
				if err := f.networkClient.AddBotToCMAccount(ctx, cmAccountAddress, cmAccountOwnerKey, botAddr); err != nil {
					return nil, fmt.Errorf("failed to add bot to CM account: %w", err)
				}
			}

			if !options.skips.ServiceRegistration {
				for _, service := range options.services {
					if err := f.networkClient.AddCMService(ctx, cmAccountAddress, cmAccountOwnerKey, service.Name, service.Fee); err != nil {
						return nil, fmt.Errorf("failed to add %s service to CM account: %w", service.Name, err)
					}
				}
			}
		}

		if !options.skips.PrefundBot {
			if err := f.networkClient.Transfer(ctx, f.networkClient.PrefundedKeys()[0], botAddr, e2eCommon.DefaultCMAccountOwnerFunds); err != nil {
				return nil, fmt.Errorf("failed to transfer funds to bot: %w", err)
			}
		}
	}

	// Prepare bot config

	port := int32(0)
	if enableRPCServer {
		port, err = f.resourceManagerSession.GetNetworkPort()
		if err != nil {
			return nil, fmt.Errorf("failed to get free port: %w", err)
		}
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	botDir := path.Join(f.dir, strconv.Itoa(len(f.bots)))

	config := &config.UnparsedConfig{
		DeveloperMode:                       true,
		E2ETestMode:                         true,
		BotKey:                              hex.EncodeToString(crypto.FromECDSA(botKey)),
		CMAccountAddress:                    cmAccountAddress.Hex(),
		ChainRPCURL:                         f.networkClient.ChainRPCURL(),
		BookingTokenAddress:                 f.networkClient.BookingTokenContractAddress().Hex(),
		NetworkFeeRecipientBotAddress:       f.asb.NetworkFeeRecipientBotAddress().Hex(),
		NetworkFeeRecipientCMAccountAddress: f.asb.NetworkFeeRecipientCMAccountAddress().Hex(),
		ChequeExpirationTime:                3600 * 24 * 30 * 7, // 7 months
		MinChequeDurationUntilExpiration:    3600 * 24 * 30 * 6, // 6 months
		CashInPeriod:                        options.cashInPeriodSeconds,
		MaxAllowedServiceFee:                "1000000000000000000", // 1 CAM
		ResponseTimeout:                     30000,                 // 30s
		PartnerPlugin: config.PartnerPluginConfig{
			Enabled:     partnerPlugin != nil,
			Host:        partnerPlugin.RPCClientConnectionString(),
			Unencrypted: true,
		},
		RPCServer: config.RPCServerConfig{
			Enabled:     enableRPCServer,
			Port:        conversion.MustInt32ToUInt64(port),
			Unencrypted: true,
		},
		Matrix: config.UnparsedMatrixConfig{Host: f.matrix.Host().String()},
		DB: config.UnparsedSQLiteDBConfig{
			DBPath: path.Join(botDir, "db"),
		},
	}

	if err := os.RemoveAll(botDir); err != nil {
		return nil, fmt.Errorf("failed to remove bot data dir: %w", err)
	}

	if err := os.MkdirAll(botDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create bot data dir: %w", err)
	}

	configPath := path.Join(botDir, "config.yaml")
	if err := e2eCommon.WriteYAMLConfig(config, configPath); err != nil {
		return nil, fmt.Errorf("failed to write bot config file: %w", err)
	}

	rpcClientConnectionString := ""
	if config.RPCServer.Enabled {
		rpcClientConnectionString = fmt.Sprintf("localhost:%d", config.RPCServer.Port) // rpc client connection string
	}

	bot := newBot(
		f.logger,
		cmAccountAddress,
		f.binPath,
		configPath,
		path.Join(botDir, "bot.log"), // log file path
		rpcClientConnectionString,
	)
	f.bots = append(f.bots, bot)

	return bot, nil
}

func (f *Factory) StopBots(ctx context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var errs []error
	errsMx := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, bot := range f.bots {
		wg.Add(1)
		go func(bot *Bot) {
			defer wg.Done()
			if err := bot.Stop(ctx); err != nil {
				errsMx.Lock()
				errs = append(errs, fmt.Errorf("failed to stop bot (%d): %w", bot.pid, err))
				errsMx.Unlock()
			}
		}(bot)
	}

	wg.Wait()
	return errors.Join(errs...)
}

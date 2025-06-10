// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	e2eCommon "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

type CMService struct {
	Name string
	Fee  int64
}

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

// Not safe for concurrent use.
type Factory struct {
	logger                 *zap.SugaredLogger
	resourceManagerSession *resources.Session
	dir                    string
	binPath                string
	networkClient          *blockchain.Client
	matrix                 *matrix.ConduitServer
	asb                    *matrix.AppService
	bots                   []*Bot
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

func (f *Factory) CreateBot(
	ctx context.Context,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	services []CMService,
	skips *Skip,
) (*Bot, chan error, error) {
	if skips == nil {
		skips = &Skip{}
	}
	var cmAccountAddress common.Address

	cmAccountOwnerKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	botKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	if !skips.CMAccountCreation {
		botAddr := crypto.PubkeyToAddress(botKey.PublicKey)
		cmAccountOwnerAddress := crypto.PubkeyToAddress(cmAccountOwnerKey.PublicKey)
		cmAccountAddress, _, err = f.networkClient.CreateCMAccount(ctx, cmAccountOwnerKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create CM account: %w", err)
		}

		if !skips.PrefundOwner {
			if err := f.networkClient.Transfer(ctx, f.networkClient.PrefundedKeys()[0], cmAccountOwnerAddress, e2eCommon.DefaultCMAccountOwnerFunds); err != nil {
				return nil, nil, fmt.Errorf("failed to transfer funds to cm account owner: %w", err)
			}

			if !skips.BotRegistration {
				if err := f.networkClient.AddBotToCMAccount(ctx, cmAccountAddress, cmAccountOwnerKey, botAddr); err != nil {
					return nil, nil, fmt.Errorf("failed to add bot to CM account: %w", err)
				}
			}

			if !skips.ServiceRegistration {
				for _, service := range services {
					if err := f.networkClient.AddCMService(ctx, cmAccountAddress, cmAccountOwnerKey, service.Name, service.Fee); err != nil {
						return nil, nil, fmt.Errorf("failed to add %s service to CM account: %w", service.Name, err)
					}
				}
			}
		}

		if !skips.PrefundBot {
			if err := f.networkClient.Transfer(ctx, f.networkClient.PrefundedKeys()[0], botAddr, e2eCommon.DefaultCMAccountOwnerFunds); err != nil {
				return nil, nil, fmt.Errorf("failed to transfer funds to bot: %w", err)
			}
		}
	}

	// Prepare bot config

	port := int32(0)
	if enableRPCServer {
		port, err = f.resourceManagerSession.GetNetworkPort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free port: %w", err)
		}
	}

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
		ChequeExpirationTime:                3600 * 24 * 30 * 7,    // 7 months
		MinChequeDurationUntilExpiration:    3600 * 24 * 30 * 6,    // 6 months
		CashInPeriod:                        3600,                  // 1h
		MaxAllowedServiceFee:                "1000000000000000000", // 1 CAM
		ResponseTimeout:                     30000,                 // 30s
		PartnerPlugin: config.PartnerPluginConfig{
			Enabled:     partnerPlugin != nil,
			Host:        partnerPlugin.Host(),
			Unencrypted: true,
		},
		RPCServer: config.RPCServerConfig{
			Enabled:     enableRPCServer,
			Port:        uint64(port),
			Unencrypted: true,
		},
		Matrix: config.UnparsedMatrixConfig{Host: f.matrix.Host().String()},
		DB: config.UnparsedSQLiteDBConfig{
			DBPath: path.Join(botDir, "db"),
		},
		Tracing: config.TracingConfig{Enabled: false},
	}

	if err := os.RemoveAll(botDir); err != nil {
		return nil, nil, fmt.Errorf("failed to remove bot data dir: %w", err)
	}

	if err := os.MkdirAll(botDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create bot data dir: %w", err)
	}

	configPath := path.Join(botDir, "config.yaml")
	if err := e2eCommon.WriteYAMLConfig(config, configPath); err != nil {
		return nil, nil, fmt.Errorf("failed to write bot config file: %w", err)
	}

	rpcClientConnectionString := ""
	if config.RPCServer.Enabled {
		rpcClientConnectionString = fmt.Sprintf("localhost:%d", config.RPCServer.Port) // rpc client connection string
	}

	// Start bot

	bot := newBot(
		f.logger,
		cmAccountAddress,
		f.binPath,
		configPath,
		path.Join(botDir, "bot.log"), // log file path
		rpcClientConnectionString,
	)
	f.bots = append(f.bots, bot)

	errChan, err := bot.Start(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start bot: %w", err)
	}

	return bot, errChan, nil
}

func (f *Factory) StopBots(ctx context.Context) error {
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

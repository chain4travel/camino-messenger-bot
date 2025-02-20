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
	"os/exec"
	"path"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-viper/mapstructure/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/readiness"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/blockchain"
	e2eGenerated "github.com/chain4travel/camino-messenger-bot/tests/e2e/bot/generated"
	e2eCommon "github.com/chain4travel/camino-messenger-bot/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/process"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/resources"
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
	migrationsDir string,
	networkClient *blockchain.Client,
	matrix *matrix.MatrixServer,
) *Factory {
	return &Factory{
		logger:                 logger,
		resourceManagerSession: resourceManagerSession,
		dir:                    path.Join(e2eTmpDir, "cmb"),
		binPath:                binPath,
		migrationsPath:         "file://" + migrationsDir,
		networkClient:          networkClient,
		matrix:                 matrix,
	}
}

// Not safe for concurrent use.
type Factory struct {
	logger                 *zap.SugaredLogger
	resourceManagerSession *resources.Session
	dir                    string
	binPath                string
	migrationsPath         string
	networkClient          *blockchain.Client
	matrix                 *matrix.MatrixServer
	bots                   []*Bot
}

func (f *Factory) CreateBot(
	ctx context.Context,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	services []CMService,
) (*Bot, chan error, error) {
	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Prepare CM account

	ownerAddr := crypto.PubkeyToAddress(key.PublicKey)

	cmAccountAddress, _, err := f.networkClient.CreateCMAccount(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CM account: %w", err)
	}

	if err := f.networkClient.Transfer(ctx, f.networkClient.PrefundedKeys()[0], ownerAddr, e2eCommon.DefaultCMAccountOwnerFunds); err != nil {
		return nil, nil, fmt.Errorf("failed to transfer funds to cm account owner: %w", err)
	}

	if err := f.networkClient.AddBotToCMAccount(ctx, cmAccountAddress, key, ownerAddr); err != nil {
		return nil, nil, fmt.Errorf("failed to add bot to CM account: %w", err)
	}

	for _, service := range services {
		if err := f.networkClient.AddCMService(ctx, cmAccountAddress, key, service.Name, service.Fee); err != nil {
			return nil, nil, fmt.Errorf("failed to add %s service to CM account: %w", service.Name, err)
		}
	}

	// Prepare bot config

	port := 0
	if enableRPCServer {
		port, err = f.resourceManagerSession.GetNetworkPort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free port: %w", err)
		}
	}

	botDir := path.Join(f.dir, strconv.Itoa(len(f.bots)))

	config := config.UnparsedConfig{
		DeveloperMode:                       true,
		BotKey:                              hex.EncodeToString(crypto.FromECDSA(key)),
		CMAccountAddress:                    cmAccountAddress.Hex(),
		ChainRPCURL:                         f.networkClient.ChainRPCURL(),
		BookingTokenAddress:                 f.networkClient.BookingTokenContractAddress().Hex(),
		NetworkFeeRecipientBotAddress:       f.matrix.NetworkFeeRecipientBotAddress().Hex(),
		NetworkFeeRecipientCMAccountAddress: f.matrix.NetworkFeeRecipientCMAccountAddress().Hex(),
		ChequeExpirationTime:                3600 * 24 * 30 * 7, // 7 months
		MinChequeDurationUntilExpiration:    3600 * 24 * 30 * 6, // 6 months
		CashInPeriod:                        3600,               // 1h
		ResponseTimeout:                     30000,              // 30s
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
			DBPath:         path.Join(botDir, "db"),
			MigrationsPath: f.migrationsPath,
		},
		Tracing: config.TracingConfig{Enabled: false},
	}

	if err := os.RemoveAll(botDir); err != nil {
		return nil, nil, fmt.Errorf("failed to remove bot data dir: %w", err)
	}

	if err := os.MkdirAll(botDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create bot data dir: %w", err)
	}

	unparsedMap := &map[string]any{}
	if err := mapstructure.Decode(config, unparsedMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config into map: %w", err)
	}
	configBytes, err := yaml.Marshal(unparsedMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal config map into yaml: %w", err)
	}

	configPath := path.Join(botDir, "config.yaml")

	if err := os.WriteFile(configPath, configBytes, 0o644); err != nil {
		return nil, nil, fmt.Errorf("failed to write config file: %w", err)
	}

	// Prepare grpc client for bot

	clientConnection, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", config.RPCServer.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	// Start bot

	cmd := exec.Command(f.binPath, "--config", configPath)

	logfile, err := os.OpenFile(path.Join(botDir, "bot.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open bot log file: %w", err)
	}
	cmd.Stdout = logfile
	cmd.Stderr = logfile

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start bot (%d): %w", cmd.Process.Pid, err)
	}

	bot := &Bot{
		logger: f.logger,
		pid:    cmd.Process.Pid,
		rpcClient: &rpcClient{
			ReadinessServiceClient: readiness.NewReadinessServiceClient(clientConnection),
			Client:                 e2eGenerated.NewClient(clientConnection),
		},
		cmAccountAddress: cmAccountAddress,
		logfile:          logfile,
	}

	// Await bot readiness

	if config.RPCServer.Enabled {
		if err := bot.awaitReady(ctx); err != nil {
			return bot, nil, fmt.Errorf("failed to await bot readiness: %w", err)
		}
	} else {
		f.logger.Warnf("bot (pid %d) started without RPC server: no readiness check", cmd.Process.Pid)
	}

	f.logger.Infof("bot (pid %d) started", cmd.Process.Pid)

	f.bots = append(f.bots, bot)

	// Await bot process error async

	errChan := make(chan error)
	go func() {
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("bot (pid %d) failed: %w", bot.pid, err)
		}
		close(errChan)
	}()

	return bot, errChan, nil
}

func (f *Factory) StopBots(ctx context.Context) error {
	var errs []error

	for _, bot := range f.bots {
		if err := bot.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop bot (%d): %w", bot.pid, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

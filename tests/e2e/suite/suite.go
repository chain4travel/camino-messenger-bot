// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package suite

import (
	"context"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminogo/secp256k1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/resources"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/runner"
)

const (
	startupTimeout  = 120 * time.Second
	shutdownTimeout = 30 * time.Second
	validatorsCount = 2
)

type Test = runner.Test[*Environment]

var (
	_ runner.BeforeRunFunc[*Environment] = (*Suite)(nil).SetupEnvironment
	_ runner.AfterRunFunc[*Environment]  = (*Suite)(nil).Cleanup
)

func New(
	nodeBinPath string,
	matrixBinPath string,
	asbBinPath string,
	partnerPluginBinPath string,
	cmbBinPath string,
	testsDataDir string,
	existingNetworkNodeURI string,
	existingNetworkAdminKey *secp256k1.PrivateKey,
	debug bool,
	filter string,
) (*Suite, error) {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.Level.SetLevel(zap.InfoLevel)
	if debug {
		zapConfig.Level.SetLevel(zap.DebugLevel)
	}
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}
	sugaredLogger := logger.Sugar()
	var testFilterElements []string
	if len(filter) > 0 {
		testFilterElements = strings.Split(filter, ",")
		if len(testFilterElements) > 0 {
			sugaredLogger.Debugf("Running only tests with the following names: %v", testFilterElements)
		}
	}

	return &Suite{
		logger:                  sugaredLogger,
		resourcesManager:        resources.NewManager(10000, 50000, 10),
		nodeBinPath:             nodeBinPath,
		matrixBinPath:           matrixBinPath,
		asbBinPath:              asbBinPath,
		partnerPluginBinPath:    partnerPluginBinPath,
		cmbBinPath:              cmbBinPath,
		testsDataDir:            testsDataDir,
		existingNetworkNodeURI:  existingNetworkNodeURI,
		existingNetworkAdminKey: existingNetworkAdminKey,
		TestFilter:              testFilterElements,
	}, nil
}

// Safe for concurrent use.
type Suite struct {
	logger                  *zap.SugaredLogger
	nodeBinPath             string
	matrixBinPath           string
	asbBinPath              string
	partnerPluginBinPath    string
	cmbBinPath              string
	testsDataDir            string
	existingNetworkNodeURI  string
	existingNetworkAdminKey *secp256k1.PrivateKey
	resourcesManager        *resources.Manager
	TestFilter              []string
}

func (s *Suite) SetupEnvironment(t *testing.T, test Test) *Environment {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	e := &Environment{
		resourceManagerSession: s.resourcesManager.NewSession(),
		Logger:                 s.logger,
	}
	test.Setup(e)

	dataDir := path.Join(s.testsDataDir, t.Name())

	var err error
	var errChan chan error
	e.networkFeeKey, err = crypto.GenerateKey()
	require.NoError(t, err)

	if len(s.existingNetworkNodeURI) > 0 {
		e.CaminoNetwork, err = blockchain.UseExistingNetwork(
			ctx,
			s.logger,
			s.existingNetworkNodeURI,
			s.existingNetworkAdminKey,
		)
		require.NoError(t, err)
	} else {
		e.CaminoNetwork, errChan, err = blockchain.StartNewNetwork(
			ctx,
			s.logger,
			e.resourceManagerSession,
			dataDir,
			s.nodeBinPath,
			validatorsCount,
		)
		require.NoError(t, err)
		common.ExpectNoErrorAsync(t, errChan)
	}

	e.ASB, errChan, err = matrix.StartNewAppService(
		ctx,
		s.logger,
		e.resourceManagerSession,
		dataDir,
		s.asbBinPath,
		e.networkFeeKey,
		e.CaminoNetwork.Client,
		e.ASBOptions...,
	)
	require.NoError(t, err)
	common.ExpectNoErrorAsync(t, errChan)

	e.matrix, errChan, err = matrix.StartNewMatrixServer(
		ctx,
		s.logger,
		e.resourceManagerSession,
		dataDir,
		s.matrixBinPath,
		e.ASB,
	)
	require.NoError(t, err)
	common.ExpectNoErrorAsync(t, errChan)

	e.partnerPluginFactory = partnerplugin.NewFactory(
		s.logger,
		e.resourceManagerSession,
		dataDir,
		s.partnerPluginBinPath,
	)

	e.botFactory = bot.NewFactory(
		s.logger,
		e.resourceManagerSession,
		dataDir,
		s.cmbBinPath,
		e.CaminoNetwork.Client,
		e.matrix,
		e.ASB,
	)

	return e
}

func (s *Suite) Cleanup(t *testing.T, e *Environment) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup

	// Components stopped without respecting the order, so they might stop with errors.
	// That will be most likely reflected in their logs, but we don't care about that at this point.
	// E.g. if network is stopped before bots, while bots have active event subscriptions,
	// bots might log errors about failed subscriptions.

	e.Logger.Debug("Stopping all services")
	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, e.botFactory.StopBots(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, e.partnerPluginFactory.StopPartnerPlugins(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, e.ASB.Stop(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, e.matrix.Stop(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, e.CaminoNetwork.Stop(ctx))
	}()

	wg.Wait()
	e.Logger.Debug("All services stopped")

	e.resourceManagerSession.ReleaseResources()
}

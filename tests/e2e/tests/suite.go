// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

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

	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

const (
	startupTimeout     = 120 * time.Second
	shutdownTimeout    = 30 * time.Second
	defaultTestTimeout = 120 * time.Second
	validatorsCount    = 2
)

func NewSuite(
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

func (s *Suite) NewTest(t *testing.T) *Test {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	tt := &Test{
		resourceManagerSession: s.resourcesManager.NewSession(),
		logger:                 s.logger,
	}

	dataDir := path.Join(s.testsDataDir, t.Name())

	var err error
	var errChan chan error
	tt.networkFeeKey, err = crypto.GenerateKey()
	require.NoError(t, err)

	if len(s.existingNetworkNodeURI) > 0 {
		tt.caminoNetwork, err = blockchain.UseExistingNetwork(
			ctx,
			s.logger,
			s.existingNetworkNodeURI,
			s.existingNetworkAdminKey,
		)
		require.NoError(t, err)
	} else {
		tt.caminoNetwork, errChan, err = blockchain.StartNewNetwork(
			ctx,
			s.logger,
			tt.resourceManagerSession,
			dataDir,
			s.nodeBinPath,
			validatorsCount,
		)
		require.NoError(t, err)
		expectNoErrorAsync(t, errChan)
	}

	tt.asb, errChan, err = matrix.StartNewAppService(
		ctx,
		s.logger,
		tt.resourceManagerSession,
		dataDir,
		s.asbBinPath,
		tt.networkFeeKey,
		tt.caminoNetwork.Client,
	)
	require.NoError(t, err)
	expectNoErrorAsync(t, errChan)

	tt.matrix, errChan, err = matrix.StartNewMatrixServer(
		ctx,
		s.logger,
		tt.resourceManagerSession,
		dataDir,
		s.matrixBinPath,
		tt.asb,
	)
	require.NoError(t, err)
	expectNoErrorAsync(t, errChan)

	tt.partnerPluginFactory = partnerplugin.NewFactory(
		s.logger,
		tt.resourceManagerSession,
		dataDir,
		s.partnerPluginBinPath,
	)

	tt.botFactory = bot.NewFactory(
		s.logger,
		tt.resourceManagerSession,
		dataDir,
		s.cmbBinPath,
		tt.caminoNetwork.Client,
		tt.matrix,
		tt.asb,
	)

	return tt
}

func (s *Suite) Cleanup(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout*10)
	defer cancel()

	var wg sync.WaitGroup

	tt.logger.Debug("Stopping all services")
	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, tt.botFactory.StopBots(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, tt.partnerPluginFactory.StopPartnerPlugins(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, tt.asb.Stop(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, tt.matrix.Stop(ctx))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, tt.caminoNetwork.Stop(ctx))
	}()

	wg.Wait()
	tt.logger.Debug("All services stopped")

	tt.resourceManagerSession.ReleaseResources()
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build e2e

package e2e

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminogo/secp256k1"
	"github.com/stretchr/testify/require"

	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/runner"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/tests"
)

const (
	flagKeyNodeBinPath          = "node"
	flagKeyMatrixBinPath        = "matrix"
	flagKeyPartnerPluginBinPath = "partner-plugin"
	flagKeyCMBBinPath           = "cmb"
	flagKeyDebug                = "debug"
	flagKeyFilter               = "filter"
)

var (
	flagNodeBinPath             string
	flagMatrixBinPath           string
	flagPartnerPluginBinPath    string
	flagCMBBinPath              string
	flagTestsDataDir            string
	flagExistingNetworkNodeURI  string
	flagExistingNetworkAdminKey string
	flagFilter                  string
	flagDebug                   bool
)

func init() {
	flag.StringVar(&flagNodeBinPath, flagKeyNodeBinPath, "", "Path to node binary.")
	flag.StringVar(&flagExistingNetworkNodeURI, "existing-network-node-uri", "", "URI of existing network node.")
	flag.StringVar(&flagExistingNetworkAdminKey, "existing-network-admin-key", "", "Admin key of existing network.")
	flag.StringVar(&flagMatrixBinPath, flagKeyMatrixBinPath, "", "Path to matrix binary.")
	flag.StringVar(&flagPartnerPluginBinPath, flagKeyPartnerPluginBinPath, "", "Path to partner plugin binary.")
	flag.StringVar(&flagCMBBinPath, flagKeyCMBBinPath, "", "Path to CMB binary.")
	flag.StringVar(&flagTestsDataDir, "tests-data-dir", "/tmp/cmb-e2e", "Path to dir with temp tests data.")
	flag.StringVar(&flagFilter, flagKeyFilter, "", "Filter for (comma separated) top level test names e.g. PingV1,AccommodationV3.")
	flag.BoolVar(&flagDebug, flagKeyDebug, false, "Debug mode")
}

func TestE2E(t *testing.T) {
	flag.Parse()
	var err error

	require.NotEmptyf(t, flagNodeBinPath, "flag -%s (node binary path) is required", flagKeyNodeBinPath)
	require.NotEmpty(t, flagMatrixBinPath, "flag -%s (matrix binary path) is required", flagKeyMatrixBinPath)
	require.NotEmpty(t, flagCMBBinPath, "flag -%s (CMB binary path) is required", flagKeyCMBBinPath)
	require.NotEmpty(t, flagPartnerPluginBinPath, "flag -%s (partner plugin binary path) is required", flagKeyPartnerPluginBinPath)

	flagNodeBinPath, err = filepath.Abs(flagNodeBinPath)
	require.NoError(t, err)
	flagMatrixBinPath, err = filepath.Abs(flagMatrixBinPath)
	require.NoError(t, err)
	flagCMBBinPath, err = filepath.Abs(flagCMBBinPath)
	require.NoError(t, err)
	flagPartnerPluginBinPath, err = filepath.Abs(flagPartnerPluginBinPath)
	require.NoError(t, err)

	checkFileExist(t, flagNodeBinPath)
	checkFileExist(t, flagMatrixBinPath)
	checkFileExist(t, flagCMBBinPath)
	checkFileExist(t, flagPartnerPluginBinPath)

	flagTestsDataDir = path.Join(flagTestsDataDir, time.Now().Format("2006-01-02_15-04-05"))

	os.RemoveAll(flagTestsDataDir)
	require.NoError(t, os.MkdirAll(flagTestsDataDir, 0o755))

	var existingNetworkAdminKey *secp256k1.PrivateKey
	if len(flagExistingNetworkAdminKey) > 0 {
		existingNetworkAdminKey = new(secp256k1.PrivateKey)
		require.NoError(t, existingNetworkAdminKey.UnmarshalText([]byte("\""+flagExistingNetworkAdminKey+"\"")))
	}

	suite, err := tests.NewSuite(
		flagNodeBinPath,
		flagMatrixBinPath,
		flagPartnerPluginBinPath,
		flagCMBBinPath,
		flagTestsDataDir,
		flagExistingNetworkNodeURI,
		existingNetworkAdminKey,
		flagDebug,
		flagFilter,
	)
	require.NoError(t, err)

	testsRunner := runner.New(
		suite.NewTest,
		suite.Cleanup,
		suite.TestFilter,
	)
	// #########################################################
	// #### Registration of the e2e test cases is done here ####
	// #########################################################
	testsRunner.Register(t, "PingV1", tests.TestPingV1)
	testsRunner.Register(t, "AccommodationV2", tests.TestAccommodationV2)
	testsRunner.Register(t, "AccommodationV3", tests.TestAccommodationV3)
	testsRunner.Register(t, "TransportV3", tests.TestTransportV3)
	testsRunner.Register(t, "BotSanity", tests.TestBotSanity)
	testsRunner.Register(t, "MintV2", tests.TestMintV2)

	maxParallelRuns := 0
	flagTestParallel := flag.Lookup("test.parallel")
	if flagTestParallel != nil {
		maxParallelRuns, err = strconv.Atoi(flagTestParallel.Value.String())
		require.NoError(t, err)
	}
	if maxParallelRuns > 1 {
		testsRunner.RunParallel(t)
	} else {
		testsRunner.Run(t)
	}
}

func checkFileExist(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.NoErrorf(t, err, "file %s does not exist", path)
}

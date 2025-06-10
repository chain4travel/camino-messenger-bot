// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminoethvm/ethadmin"
	"github.com/chain4travel/caminogoeth-compat/caminogo/config"
	"github.com/chain4travel/caminogoeth-compat/caminogo/genesis"
	"github.com/chain4travel/caminogoeth-compat/caminogo/secp256k1"
	"github.com/chain4travel/caminogoeth-compat/caminogo/staking"
	"github.com/chain4travel/caminogoeth-compat/caminogo/units"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	e2eCommon "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/process"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

const (
	NetworkID = 1005

	defaultPFunds = 1_000_000 * units.Avax
	defaultXFunds = 1_000_000 * units.Avax
	defaultCFunds = 1_000_000 * units.Avax
)

var (
	evmChainID  = big.NewInt(502)
	evmChainIDs = map[uint32]*big.Int{
		NetworkID: evmChainID,
	}
	evmAdminContract        = common.HexToAddress("0x010000000000000000000000000000000000000a")
	evmAddrStateKYCVerified = big.NewInt(ethadmin.KYC_VERIFIED)
	defaultCFundsBig        = big.NewInt(0).Mul(big.NewInt(0).SetUint64(defaultCFunds), e2eCommon.X2CRateBig)
)

func StartNewNetwork(
	ctx context.Context,
	logger *zap.SugaredLogger,
	resourceManagerSession *resources.Session,
	dataDir string,
	nodeBinPath string,
	validatorsCount int,
) (*Network, chan error, error) {
	logger.Debugf("Starting blockchain network (%d validators)...", validatorsCount)

	var err error
	httpPorts := make([]int32, validatorsCount)
	stakingPorts := make([]int32, validatorsCount)
	for i := 0; i < validatorsCount; i++ {
		httpPorts[i], err = resourceManagerSession.GetNetworkPort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free port: %w", err)
		}
		stakingPorts[i], err = resourceManagerSession.GetNetworkPort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get free port: %w", err)
		}
	}

	networkDir := path.Join(dataDir, "blockchain")
	if err := os.RemoveAll(networkDir); err != nil {
		return nil, nil, fmt.Errorf("failed to remove blockchain tmp dir: %w", err)
	}

	secp256k1Factory := secp256k1.Factory{}
	bootstrapIDs := make([]string, validatorsCount)
	bootstrapIPs := make([]string, validatorsCount)
	validators := make([]validator, validatorsCount)
	for i := 0; i < validatorsCount; i++ {
		key, err := secp256k1Factory.NewPrivateKey()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate key: %w", err)
		}

		keyBytes, certBytes, nodeID, err := staking.NewCertAndKeyBytesWithSecpKey()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to init node staking key pair: %w", err)
		}

		keyPath := path.Join(networkDir, nodeID.String(), "staker.key")
		certPath := path.Join(networkDir, nodeID.String(), "staker.crt")

		if err := staking.StoreStakingKeyPair(keyPath, certPath, keyBytes, certBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to load tls cert from files: %w", err)
		}

		validators[i] = validator{owner: key, nodeID: nodeID}
		bootstrapIDs[i] = nodeID.String()
		bootstrapIPs[i] = fmt.Sprintf("127.0.0.1:%d", stakingPorts[i])
	}

	adminKey, err := secp256k1Factory.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}

	funds := make([]allocation, 10)
	prefundedKeys := make([]*secp256k1.PrivateKey, 10)
	prefundedECDSAKeys := make([]*ecdsa.PrivateKey, 10)
	for i := range prefundedKeys {
		prefundedKeys[i], err = secp256k1Factory.NewPrivateKey()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate key: %w", err)
		}
		funds[i] = allocation{
			owner: prefundedKeys[i],
			addrStates: genesis.AddressStates{
				KYCVerified: true,
			},
			p: defaultPFunds,
			x: defaultXFunds,
			c: defaultCFundsBig,
		}
		prefundedECDSAKeys[i] = prefundedKeys[i].ToECDSA()
	}

	genesis, err := generateGenesis(time.Now(), adminKey, validators, funds, NetworkID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create genesis: %w", err)
	}

	n := &Network{
		logger:          logger,
		genesis:         genesis,
		bootstrapIDsArg: strings.Join(bootstrapIDs, ","),
		bootstrapIPsArg: strings.Join(bootstrapIPs, ","),
		prefundedKeys:   prefundedECDSAKeys,
		networkName:     strconv.Itoa(NetworkID),
		nodeBinPath:     nodeBinPath,
		evmAdminKey:     adminKey.ToECDSA(),
		nodes:           make([]*Node, validatorsCount),
	}

	errChans := make([]chan error, validatorsCount)

	errGroup, gCtx := errgroup.WithContext(ctx)

	for i := range validators {
		bootstrapIDs := slices.Delete(slices.Clone(bootstrapIDs), i, i+1)
		boottstrapIPs := slices.Delete(slices.Clone(bootstrapIPs), i, i+1)

		errGroup.Go(func() error {
			n.nodes[i], errChans[i], err = n.startNewNode(
				gCtx,
				path.Join(networkDir, validators[i].nodeID.String(), "node"),
				path.Join(networkDir, validators[i].nodeID.String(), "staker.key"),
				path.Join(networkDir, validators[i].nodeID.String(), "staker.crt"),
				httpPorts[i],
				stakingPorts[i],
				strings.Join(bootstrapIDs, ","),
				strings.Join(boottstrapIPs, ","),
				i,
			)
			if err != nil {
				return fmt.Errorf("failed to start node (%d): %w", i, err)
			}
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return n, nil, fmt.Errorf("failed to start nodes: %w", err)
	}

	n.Client = n.nodes[0].client

	logger.Debug("Preparing EVM admin...")

	if err := n.Client.prepareAdmin(ctx); err != nil {
		return n, nil, fmt.Errorf("failed to prepare EVM admin: %w", err)
	}

	logger.Debug("Preparing CMB contracts...")

	if err := n.Client.prepareCMBContracts(ctx); err != nil {
		return n, nil, fmt.Errorf("failed to prepare CMB contracts: %w", err)
	}

	logger.Debug("Blockchain network started")

	return n, combineErrChannels(errChans...), nil
}

func UseExistingNetwork(
	ctx context.Context,
	logger *zap.SugaredLogger,
	nodeURI string,
	adminKey *secp256k1.PrivateKey,
) (*Network, error) {
	logger.Info("Connecting to existing network...")

	client, err := newClient(ctx, nodeURI, nil, adminKey.ToECDSA())
	if err != nil {
		return nil, fmt.Errorf("failed to create node client: %w", err)
	}

	n := &Network{
		evmAdminKey: adminKey.ToECDSA(),
		Client:      client,
	}

	logger.Info("Preparing EVM admin...")

	if err := n.Client.prepareAdmin(ctx); err != nil {
		return n, fmt.Errorf("failed to prepare EVM admin: %w", err)
	}

	logger.Info("Preparing CMB contracts...")

	if err := n.Client.prepareCMBContracts(ctx); err != nil {
		return n, fmt.Errorf("failed to prepare CMB contracts: %w", err)
	}

	logger.Info("Blockchain network started")

	return n, nil
}

// Not safe for concurrent use.
type Network struct {
	Client *Client

	logger          *zap.SugaredLogger
	networkName     string
	genesis         string
	nodes           []*Node
	nodeBinPath     string
	bootstrapIDsArg string
	bootstrapIPsArg string
	prefundedKeys   []*ecdsa.PrivateKey
	evmAdminKey     *ecdsa.PrivateKey
}

func (n *Network) startNewNode(
	ctx context.Context,
	nodeDir string,
	stakerKeyPath string,
	stakerCertPath string,
	httpPort int32,
	stakingPort int32,
	bootstrapIDsArg string,
	bootstrapIPsArg string,
	nodeIndex int,
) (*Node, chan error, error) {
	n.logger.Debugf("Starting blockchain node %d (http port %d)...", nodeIndex, httpPort)

	if err := os.RemoveAll(nodeDir); err != nil {
		return nil, nil, fmt.Errorf("failed to remove blockchain node tmp dir: %w", err)
	}

	cmd := exec.Command(n.nodeBinPath, //nolint:gosec // this is a caminogo node binary, not some injection.
		fmt.Sprintf("--%s=%d", config.HTTPPortKey, httpPort),
		fmt.Sprintf("--%s=%d", config.StakingPortKey, stakingPort),
		fmt.Sprintf("--%s=%s", config.StakingTLSKeyPathKey, stakerKeyPath),
		fmt.Sprintf("--%s=%s", config.StakingCertPathKey, stakerCertPath),
		fmt.Sprintf("--%s=%s", config.DBPathKey, path.Join(nodeDir, "db")),
		fmt.Sprintf("--%s=%s", config.DataDirKey, path.Join(nodeDir, "data")),
		fmt.Sprintf("--%s=%s", config.GenesisConfigContentKey, n.genesis),
		fmt.Sprintf("--%s=%s", config.BootstrapIDsKey, bootstrapIDsArg),
		fmt.Sprintf("--%s=%s", config.BootstrapIPsKey, bootstrapIPsArg),
		fmt.Sprintf("--%s=%s", config.NetworkNameKey, n.networkName),
	)

	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create node directory: %w", err)
	}
	logFile, err := os.OpenFile(path.Join(nodeDir, fmt.Sprintf("node-%d.log", nodeIndex)), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create node log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start node (%d): %w", cmd.Process.Pid, err)
	}

	node := &Node{
		logger:  n.logger,
		pid:     cmd.Process.Pid,
		nodeURI: fmt.Sprintf("localhost:%d", httpPort),
		logFile: logFile,
	}

	// health check will fail if its 1 node network, so we're doing bootstrap check
	if err := node.awaitBootstrapped(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to wait for node (%d) to be bootstrapped: %w", cmd.Process.Pid, err)
	}

	client, err := newClient(ctx, node.nodeURI, n.prefundedKeys, n.evmAdminKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create node client: %w", err)
	}
	node.client = client

	n.logger.Debugf("Blockchain node %d (pid %d) started", nodeIndex, node.pid)

	errChan := make(chan error)
	go func() {
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("blockchain node (pid %d) failed: %w", node.pid, err)
		}
		close(errChan)
	}()

	return node, errChan, nil
}

func (n *Network) Stop(ctx context.Context) error {
	if n == nil {
		return nil
	}

	var errs []error
	errsMx := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, node := range n.nodes {
		wg.Add(1)
		go func(node *Node) {
			defer wg.Done()
			if err := node.Stop(ctx); err != nil {
				errsMx.Lock()
				errs = append(errs, fmt.Errorf("failed to stop node: %w", err))
				errsMx.Unlock()
			}
		}(node)
	}

	wg.Wait()

	if len(errs) == 0 {
		n.logger.Debug("Blockchain network stopped successfully")
	}
	return errors.Join(errs...)
}

func combineErrChannels(errChans ...chan error) chan error {
	wg := sync.WaitGroup{}
	out := make(chan error)

	forward := func(c <-chan error) {
		defer wg.Done()
		for err := range c {
			out <- err
		}
	}

	wg.Add(len(errChans))
	for _, c := range errChans {
		go forward(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

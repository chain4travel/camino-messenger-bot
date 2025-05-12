// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/process"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

const matrixRequestTickerInterval = 1000 * time.Millisecond

func init() {
	os.Setenv("CONDUIT_CONFIG", "")
	os.Setenv("CONDUIT_SERVER_NAME", "localhost")
	os.Setenv("CONDUIT_DATABASE_BACKEND", "rocksdb")
	// os.Setenv("CONDUIT_MAX_REQUEST_SIZE", "20_000_000")
	os.Setenv("CONDUIT_ALLOW_REGISTRATION", "false")
	os.Setenv("CONDUIT_ALLOW_FEDERATION", "false")
	os.Setenv("CONDUIT_ALLOW_CHECK_FOR_UPDATES", "false")
	os.Setenv("CONDUIT_TRUSTED_SERVERS", "[]")
	// os.Setenv("CONDUIT_MAX_CONCURRENT_REQUESTS", "100")
	// os.Setenv("CONDUIT_LOG", "debug,rocket=off,_=off,sled=off")
	os.Setenv("CONDUIT_ADDRESS", "0.0.0.0")
	os.Setenv("CONDUIT_LOG", "trace,rocket=off,_=off,sled=off")
	os.Setenv("CONDUIT_CAMINO_NETWORK_ID", fmt.Sprintf("%d", blockchain.NetworkID))
}

func StartNewMatrixServer(
	ctx context.Context,
	logger *zap.SugaredLogger,
	resourceManagerSession *resources.Session,
	dataDir string,
	matrixBinPath string,
	appService *AppService,
) (*ConduitServer, chan error, error) {
	logger.Debug("Starting matrix server...")

	matrixDir := path.Join(dataDir, "matrix")
	if err := os.RemoveAll(matrixDir); err != nil {
		return nil, nil, fmt.Errorf("failed to remove matrix server tmp dir: %w", err)
	}

	port, err := resourceManagerSession.GetNetworkPort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get free port: %w", err)
	}

	client, err := mautrix.NewClient(fmt.Sprintf("http://localhost:%d", port), "", "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create matrix client: %w", err)
	}

	dbDir := path.Join(matrixDir, "db")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create matrix server db dir: %w", err)
	}

	cmd := exec.Command(matrixBinPath)
	cmd.Env = append(os.Environ(),
		"CONDUIT_DATABASE_PATH="+dbDir,
		"CONDUIT_PORT="+strconv.Itoa(int(port)),
		"CONDUIT_CAMINO_APP_SERVICE_URL="+appService.Host(),
		"CONDUIT_CAMINO_APP_SERVICE_HS_TOKEN="+appService.HSAccessToken(),
		"CONDUIT_CAMINO_APP_SERVICE_AS_TOKEN="+appService.ASAccessToken(),
	)

	logFile, err := os.OpenFile(matrixDir+"/conduit-server.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create conduit server log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start matrix server (%d): %w", cmd.Process.Pid, err)
	}

	m := &ConduitServer{
		logger:    logger,
		pid:       cmd.Process.Pid,
		matrixDir: matrixDir,
		client:    client,
		logFile:   logFile,
	}

	if err := m.awaitReady(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to wait for matrix (%d) to be ready: %w", cmd.Process.Pid, err)
	}

	logger.Debugf("Matrix server (pid %d) started", cmd.Process.Pid)

	errChan := make(chan error)
	go func() {
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("matrix server (pid %d) failed: %w", m.pid, err)
		}
		close(errChan)
	}()

	return m, errChan, nil
}

// Conduit server.
// Not safe for concurrent use.
type ConduitServer struct {
	logger    *zap.SugaredLogger
	pid       int
	matrixDir string
	client    *mautrix.Client
	logFile   *os.File
}

func (m *ConduitServer) Host() *url.URL {
	return m.client.HomeserverURL
}

func (m *ConduitServer) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}

	if err := process.StopProcess(ctx, m.pid); err != nil {
		return fmt.Errorf("failed to stop matrix server process with pid %d: %w", m.pid, err)
	}
	if err := m.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close matrix server logFile: %w", err)
	}
	m.logger.Debugf("Matrix server (pid %d) stopped", m.pid)
	return nil
}

// Will try to do periodic login
func (m *ConduitServer) awaitReady(ctx context.Context) error {
	cryptoHelper, err := cryptohelper.NewCryptoHelper(m.client, []byte("meow"), path.Join(m.matrixDir, "client-db"))
	if err != nil {
		return fmt.Errorf("failed to create matrix crypto helper: %w", err)
	}

	randomKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate random key for matrix login: %w", err)
	}

	signature, message, err := matrix.SignPublicKey(randomKey)
	if err != nil {
		return err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	ticker := time.NewTicker(matrixRequestTickerInterval)
	defer ticker.Stop()

	for {
		if err := cryptoHelper.Init(ctx); err == nil {
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

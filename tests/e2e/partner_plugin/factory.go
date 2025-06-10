// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	ppmock "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/server"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/process"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

func NewFactory(
	logger *zap.SugaredLogger,
	resourceManagerSession *resources.Session,
	e2eTmpDir string,
	binPath string,
) *Factory {
	return &Factory{
		logger:                 logger,
		resourceManagerSession: resourceManagerSession,
		binPath:                binPath,
		dir:                    path.Join(e2eTmpDir, "pp-mock"),
	}
}

// Not safe for concurrent use.
type Factory struct {
	logger                 *zap.SugaredLogger
	resourceManagerSession *resources.Session
	dir                    string
	binPath                string
	partnerPlugins         []*PartnerPlugin
}

func (f *Factory) PartnerPluginsCount() int {
	return len(f.partnerPlugins)
}

func (f *Factory) CreatePartnerPlugin(ctx context.Context) (*PartnerPlugin, chan error, error) {
	port, err := f.resourceManagerSession.GetNetworkPort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get free port: %w", err)
	}

	host := fmt.Sprintf("localhost:%d", port)

	clientConnection, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	hostURL, err := url.Parse(fmt.Sprintf("gprc://%s", host))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse host url: %w", err)
	}

	cmd := exec.Command(f.binPath) //nolint:gosec // this is a partner plugin mock binary, not some injection.
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("%s=%d", ppmock.EnvKeyPort, port),
		fmt.Sprintf("%s=true", ppmock.EnvKeyEventsEnabled),
		fmt.Sprintf("%s=true", ppmock.EnvE2ETestMode),
	)

	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create pp-mock directory: %w", err)
	}
	logFileName := path.Join(f.dir, fmt.Sprintf("partner-plugin-%d.log", port))
	logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open pp-mock log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start partner-plugin (%d): %w", cmd.Process.Pid, err)
	}

	pp := &PartnerPlugin{
		logger:       f.logger,
		pid:          cmd.Process.Pid,
		host:         hostURL,
		pingClient:   pingv1grpc.NewPingServiceClient(clientConnection),
		eventsClient: events.NewEventsServiceClient(clientConnection),
		logFile:      logFile,
	}

	if err := pp.awaitReady(ctx); err != nil {
		return pp, nil, fmt.Errorf("failed to wait for partner-plugin to be ready: %w", err)
	}

	f.logger.Debugf("Partner-plugin (pid %d) started", cmd.Process.Pid)

	f.partnerPlugins = append(f.partnerPlugins, pp)

	errChan := make(chan error)
	go func() {
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("partner-plugin (pid %d) failed: %w", pp.pid, err)
		}
		close(errChan)
	}()

	return pp, errChan, nil
}

func (f *Factory) StopPartnerPlugins(ctx context.Context) error {
	var errs []error
	errsMx := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, pp := range f.partnerPlugins {
		wg.Add(1)
		go func(pp *PartnerPlugin) {
			defer wg.Done()
			if err := pp.Stop(ctx); err != nil {
				errsMx.Lock()
				errs = append(errs, fmt.Errorf("failed to stop partner plugin (%d): %w", pp.pid, err))
				errsMx.Unlock()
			}
		}(pp)
	}

	wg.Wait()
	return errors.Join(errs...)
}

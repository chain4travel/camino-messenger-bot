// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/proto/pb/events"
	ppmock "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/server"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/process"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const requestTickerInterval = 500 * time.Millisecond

type PartnerPlugin struct {
	mutex  sync.Mutex
	logger *zap.SugaredLogger

	binPath             string
	port                int32
	logPath             string
	rpcConnectionString string

	pid          int
	logFile      *os.File
	pingClient   pingv1grpc.PingServiceClient
	eventsClient events.EventsServiceClient
}

func newPartnerPlugin(
	logger *zap.SugaredLogger,
	binPath string,
	port int32,
	logPath string,
) *PartnerPlugin {
	return &PartnerPlugin{
		logger:              logger,
		binPath:             binPath,
		port:                port,
		logPath:             logPath,
		rpcConnectionString: fmt.Sprintf("localhost:%d", port),
	}
}

func (pp *PartnerPlugin) RPCClientConnectionString() string {
	if pp == nil {
		return ""
	}
	return pp.rpcConnectionString
}

func (pp *PartnerPlugin) Start(ctx context.Context) (chan error, error) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	// Prepare log file for partner-plugin

	logFile, err := os.OpenFile(pp.logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open partner-plugin log file: %w", err)
	}
	pp.logFile = logFile

	// Prepare cmd and start partner-plugin process

	cmd := exec.Command(pp.binPath) //nolint:gosec,noctx // this is a partner plugin binary, not some injection.
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("%s=%d", ppmock.EnvKeyPort, pp.port),
		fmt.Sprintf("%s=true", ppmock.EnvKeyEventsEnabled),
		fmt.Sprintf("%s=true", ppmock.EnvE2ETestMode),
	)
	cmd.Stdout = pp.logFile
	cmd.Stderr = pp.logFile

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start partner-plugin (%d): %w", cmd.Process.Pid, err)
	}

	pp.pid = cmd.Process.Pid

	// Prepare RPC client

	rpcClientConnection, err := grpc.NewClient(pp.rpcConnectionString, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}
	pp.pingClient = pingv1grpc.NewPingServiceClient(rpcClientConnection)
	pp.eventsClient = events.NewEventsServiceClient(rpcClientConnection)

	// Await partner-plugin readiness if RPC client is set

	if err := pp.awaitReady(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for partner-plugin to be ready: %w", err)
	}

	// Await partner-plugin process error async

	errChan := make(chan error)
	go func() {
		defer close(errChan)
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("partner-plugin (pid %d) failed: %w", pp.pid, err)
		}
	}()

	// Successfully started partner-plugin

	pp.logger.Debugf("Partner-plugin (pid %d) started", cmd.Process.Pid)
	return errChan, nil
}

func (pp *PartnerPlugin) Stop(ctx context.Context) error {
	if pp == nil {
		return nil
	}

	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	if err := process.StopProcess(ctx, pp.pid); err != nil {
		return fmt.Errorf("failed to stop partner plugin process with pid %d: %w", pp.pid, err)
	}
	pp.logger.Debugf("Partner plugin (pid %d) stopped", pp.pid)
	if err := pp.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close partner plugin logFile: %w", err)
	}
	return nil
}

func (pp *PartnerPlugin) SubscribeForEvents(ctx context.Context) (events.EventsService_SubscribeClient, error) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	return pp.eventsClient.Subscribe(ctx, &emptypb.Empty{})
}

func (pp *PartnerPlugin) awaitReady(ctx context.Context) error {
	ticker := time.NewTicker(requestTickerInterval)
	defer ticker.Stop()

	for {
		res, err := pp.pingClient.Ping(ctx, &pingv1.PingRequest{
			Header: &typesv1.RequestHeader{
				BaseHeader: &typesv1.Header{
					Version:              &typesv1.Version{},
					EndUserWalletAddress: ethCommon.Address{}.Hex(),
				},
			},
			PingMessage: common.PingMessage,
			Timestamp:   timestamppb.Now(),
		})
		if err == nil && res.Header.Status == typesv1.StatusType_STATUS_TYPE_SUCCESS {
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

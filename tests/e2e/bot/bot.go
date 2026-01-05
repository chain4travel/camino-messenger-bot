// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package bot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/v12/proto/pb/readiness"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/process"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const requestTickerInterval = 500 * time.Millisecond

func newBot(
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	binPath string,
	configPath string,
	logPath string,
	rpcConnectionString string,
) *Bot {
	return &Bot{
		logger:              logger,
		cmAccountAddress:    cmAccountAddress,
		binPath:             binPath,
		configPath:          configPath,
		logPath:             logPath,
		rpcConnectionString: rpcConnectionString,
	}
}

type Bot struct {
	logger *zap.SugaredLogger
	mutex  sync.Mutex

	cmAccountAddress    common.Address
	binPath             string
	configPath          string
	logPath             string
	rpcConnectionString string

	pid     int
	logFile *os.File

	*rpcClient
}

func (b *Bot) Start(ctx context.Context) (chan error, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.start(ctx)
}

func (b *Bot) start(ctx context.Context) (chan error, error) {
	// Prepare log file for bot

	logFile, err := os.OpenFile(b.logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open bot log file: %w", err)
	}
	b.logFile = logFile

	// Prepare cmd and start bot process

	cmd := exec.Command(b.binPath, "--config", b.configPath) //nolint:gosec,noctx // this is a cmb binary, not some injection.
	cmd.Stdout = b.logFile
	cmd.Stderr = b.logFile

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start bot (%d): %w", cmd.Process.Pid, err)
	}

	b.pid = cmd.Process.Pid

	// Prepare RPC client

	if b.rpcConnectionString != "" {
		rpcClientConnection, err := grpc.NewClient(b.rpcConnectionString, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("failed to create grpc client: %w", err)
		}
		b.rpcClient = &rpcClient{
			connection:             rpcClientConnection,
			ReadinessServiceClient: readiness.NewReadinessServiceClient(rpcClientConnection),
			Client:                 generated.NewClient(rpcClientConnection),
		}
	}

	// Await bot readiness if RPC client is set

	if b.rpcClient != nil {
		if err := b.awaitReady(ctx); err != nil {
			return nil, fmt.Errorf("failed to await bot readiness (pid %d): %w", b.pid, err)
		}
	} else {
		b.logger.Debugf("bot (pid %d) started without RPC server: no readiness check", cmd.Process.Pid)
	}

	// Await bot process error async

	errChan := make(chan error)
	go func() {
		defer close(errChan)
		err := <-process.ListenForProcessError(cmd)
		if err != nil {
			errChan <- fmt.Errorf("bot (pid %d) failed: %w", cmd.Process.Pid, err)
		}
	}()

	// Successfully started bot

	b.logger.Debugf("bot (pid %d) started", cmd.Process.Pid)
	return errChan, nil
}

func (b *Bot) Stop(ctx context.Context) error {
	if b == nil {
		return nil
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.stop(ctx)
}

func (b *Bot) stop(ctx context.Context) error {
	g := errgroup.Group{}
	processStopped := make(chan struct{})
	pid := b.pid

	g.Go(func() error {
		defer close(processStopped)
		if err := process.StopProcess(ctx, b.pid); err != nil {
			err := fmt.Errorf("failed to stop bot process: %w", err)
			b.logger.Error(err)
			return err
		}
		b.pid = 0
		b.logger.Debugf("Bot process (pid %d) stopped", pid)
		return nil
	})

	if b.logFile != nil {
		g.Go(func() error {
			select {
			case <-processStopped:
			case <-ctx.Done():
				return ctx.Err()
			}
			if err := b.logFile.Close(); err != nil {
				return fmt.Errorf("failed to close bot logFile: %w", err)
			}
			b.logFile = nil
			return nil
		})
	}

	if b.rpcClient != nil {
		g.Go(func() error {
			if err := b.close(); err != nil {
				return fmt.Errorf("failed to close rpc client: %w", err)
			}
			b.rpcClient = nil
			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		err = fmt.Errorf("failed to stop bot (pid %d) or close its resources: %w", b.pid, err)
		b.logger.Error(err)
	}
	return err
}

func (b *Bot) Restart(ctx context.Context) (chan error, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.logger.Debugf("Restarting bot (pid %d)", b.pid)

	oldPID := b.pid

	if err := b.stop(ctx); err != nil {
		return nil, err
	}

	errChan, err := b.start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start bot (old pid %d, new pid %d): %w", oldPID, b.pid, err)
	}

	b.logger.Debugf("Bot (old pid %d, new pid %d) restarted", oldPID, b.pid)
	return errChan, nil
}

func (b *Bot) CMAccountAddress() common.Address {
	return b.cmAccountAddress
}

func (b *Bot) awaitReady(ctx context.Context) error {
	ticker := time.NewTicker(requestTickerInterval)
	defer ticker.Stop()

	for {
		res, err := b.Readiness(ctx, &emptypb.Empty{})
		if err == nil && res.Status == server.StatusReady {
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type rpcClient struct {
	connection *grpc.ClientConn

	readiness.ReadinessServiceClient
	*generated.Client
}

func (c *rpcClient) close() error {
	return c.connection.Close()
}

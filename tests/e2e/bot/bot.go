// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package bot

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/chain4travel/camino-messenger-bot/internal/rpc/server"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/readiness"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot/generated"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/process"
	"github.com/ethereum/go-ethereum/common"
)

const requestTickerInterval = 500 * time.Millisecond

type Bot struct {
	logger           *zap.SugaredLogger
	pid              int
	cmAccountAddress common.Address
	logfile          *os.File

	*rpcClient
}

func (b *Bot) Stop(ctx context.Context) error {
	if b == nil {
		return nil
	}
	if err := process.StopProcess(ctx, b.pid); err != nil {
		return fmt.Errorf("failed to stop cmb process with pid %d: %w", b.pid, err)
	}
	b.logger.Infof("Bot (pid %d) stopped", b.pid)
	if err := b.logfile.Close(); err != nil {
		return fmt.Errorf("failed to close partner plugin logfile: %w", err)
	}
	return nil
}

func (b *Bot) CMAccountAddress() common.Address {
	return b.cmAccountAddress
}

func (b *Bot) awaitReady(ctx context.Context) error {
	ticker := time.NewTicker(requestTickerInterval)
	defer ticker.Stop()

	for {
		res, err := b.rpcClient.Readiness(ctx, &emptypb.Empty{})
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
	readiness.ReadinessServiceClient
	*generated.Client
}

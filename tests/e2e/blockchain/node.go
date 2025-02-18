// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminogo/info"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/tests/e2e/process"
)

const nodeRequestTickerInterval = 500 * time.Millisecond

type Node struct {
	logger  *zap.SugaredLogger
	pid     int
	client  *Client
	nodeURI string
}

func (n *Node) Stop(ctx context.Context) error {
	if err := process.StopProcess(ctx, n.pid); err != nil {
		return fmt.Errorf("failed to stop node process with pid %d: %w", n.pid, err)
	}
	n.logger.Infof("Blockchain node (pid %d) stopped", n.pid)
	return nil
}

func (n *Node) awaitBootstrapped(ctx context.Context) error {
	client := info.NewClient(n.nodeURI)
	ticker := time.NewTicker(nodeRequestTickerInterval)
	defer ticker.Stop()

	for {
		res, err := client.IsBootstrapped(ctx, "C")
		if err == nil && res {
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

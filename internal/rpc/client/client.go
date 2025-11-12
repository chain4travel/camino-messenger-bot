// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"fmt"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/chain4travel/camino-messenger-bot/v12/config"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/utils/tls"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RPCClient struct {
	logger     *zap.SugaredLogger
	ClientConn *grpc.ClientConn
}

func NewClient(cfg config.PartnerPluginConfig, logger *zap.SugaredLogger) (*RPCClient, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var opts []grpc.DialOption
	if cfg.Unencrypted {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsCreds, err := tls.LoadCATLSCredentials(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("could not load TLS keys: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	}

	clientConnection, err := grpc.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, err
	}

	return &RPCClient{
		logger:     logger,
		ClientConn: clientConnection,
	}, nil
}

func (rc *RPCClient) Shutdown() error {
	return rc.ClientConn.Close()
}

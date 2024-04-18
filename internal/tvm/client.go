/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tvm

import (
	"context"
	"errors"
	"fmt"
	"github.com/ava-labs/hypersdk/codec"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/pubsub"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/caminotravelvm/auth"
	"github.com/chain4travel/caminotravelvm/cmd/caminotravel-cli/cmd"
	brpc "github.com/chain4travel/caminotravelvm/rpc"
)

var (
	ErrAwaitTxConfirmationTimeout = errors.New("awaiting transaction confirmation exceeded timeout of")
)

type Client struct {
	cli         *rpc.JSONRPCClient
	ws          *rpc.WebSocketClient
	tCli        *brpc.JSONRPCClient
	authFactory *auth.SECP256K1Factory
	pk          *secp256k1.PrivateKey

	Timeout time.Duration // milliseconds
}

func (c *Client) SendTxAndWait(ctx context.Context, action chain.Action) (bool, ids.ID, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	return cmd.SendAndWait(ctxWithTimeout, nil, action, c.cli, c.ws, c.tCli, c.authFactory, false)
}

func NewClient(cfg config.TvmConfig) (*Client, error) {
	uri := fmt.Sprintf("%s/ext/bc/%s", cfg.NodeURI, cfg.ChainID)
	chainID, err := ids.FromString(cfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chainID: %w", err)
	}

	cli := rpc.NewJSONRPCClient(uri)
	ws, err := rpc.NewWebSocketClient(uri, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket client: %w", err)
	}
	tCli := brpc.NewJSONRPCClient(
		uri,
		uint32(cfg.NetworkID),
		chainID,
	)
	pk, err := readPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	factory := auth.NewSECP256K1Factory(*pk)

	return &Client{
		cli:         cli,
		ws:          ws,
		tCli:        tCli,
		authFactory: factory,
		pk:          pk,
		Timeout:     time.Duration(cfg.AwaitTxConfirmationTimeout) * time.Millisecond,
	}, nil
}

func (c *Client) Address() codec.Address {
	return auth.NewSECP256K1Address(*c.pk.PublicKey())
}

func readPrivateKey(keyStr string) (*secp256k1.PrivateKey, error) {
	key := new(secp256k1.PrivateKey)
	if err := key.UnmarshalText([]byte("\"" + keyStr + "\"")); err != nil {
		return nil, err
	}
	return key, nil
}

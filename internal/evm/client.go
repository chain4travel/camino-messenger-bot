/*
 * Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package evm

import (
	"errors"

	//"fmt"

	//"github.com/ava-labs/hypersdk/pubsub"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum/ethclient"
)

var ErrAwaitTxConfirmationTimeout = errors.New("awaiting transaction confirmation exceeded timeout of")

/*
type Client struct {
	cli *rpc.JSONRPCClient
	ws  *rpc.WebSocketClient
	//tCli        *brpc.JSONRPCClient
	authFactory *auth.SECP256K1Factory
	pk          *secp256k1.PrivateKey

	Timeout time.Duration // milliseconds
}
*/
/*
func (c *Client) SendTxAndWait(ctx context.Context, action chain.Action) (bool, ids.ID, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	return cmd.SendAndWait(ctxWithTimeout, nil, action, c.cli, c.ws, c.tCli, c.authFactory, false)
} */

func NewClient(cfg config.EvmConfig) (*ethclient.Client, error) {
	return ethclient.Dial(cfg.RpcURL)
}

/*
	func NewClient(cfg config.EvmConfig) (*Client, error) {
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
*/

/*
	 func (c *Client) Address() codec.Address {
		return auth.NewSECP256K1Address(*c.pk.PublicKey())
	}
*/
/* func readPrivateKey(keyStr string) (*secp256k1.PrivateKey, error) {
	key := new(secp256k1.PrivateKey)
	if err := key.UnmarshalText([]byte("\"" + keyStr + "\"")); err != nil {
		return nil, err
	}
	return key, nil
} */

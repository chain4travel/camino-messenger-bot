//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

package evm

import (
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum/ethclient"
)

func NewClient(cfg config.EvmConfig) (*ethclient.Client, error) {
	return ethclient.Dial(cfg.RPCURL)
}

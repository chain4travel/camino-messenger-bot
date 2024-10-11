/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package cmaccountscache

import (
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/common"
)

var _ CMAccountsCache = (*NoopCMAccountsCache)(nil)

type NoopCMAccountsCache struct{}

func (NoopCMAccountsCache) Get(common.Address) (*cmaccount.Cmaccount, error) {
	return nil, nil
}

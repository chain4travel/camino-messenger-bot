/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	cmaccountscache "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/common"
)

var _ cmaccountscache.CMAccountsCache = (*noopCMAccountsCache)(nil)

type noopCMAccountsCache struct{}

func (noopCMAccountsCache) Get(common.Address) (*cmaccount.Cmaccount, error) {
	return nil, nil
}

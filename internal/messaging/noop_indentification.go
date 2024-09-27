/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"github.com/ethereum/go-ethereum/common"
	"maunium.net/go/mautrix/id"
)

var _ IdentificationHandler = (*NoopIdentification)(nil)

type NoopIdentification struct{}

func (NoopIdentification) getAllBotAddressesFromCMAccountAddress(common.Address) ([]common.Address, error) {
	return nil, nil
}

func (NoopIdentification) getFirstBotFromCMAccountAddress(common.Address) (common.Address, error) {
	return common.Address{}, nil
}

func (NoopIdentification) getMyCMAccountAddress() common.Address {
	return common.Address{}
}

func (NoopIdentification) getFirstBotUserIDFromCMAccountAddress(_ common.Address) (id.UserID, error) {
	return "", nil
}

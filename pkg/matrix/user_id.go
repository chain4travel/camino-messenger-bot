// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"maunium.net/go/mautrix/id"
)

func UserIDFromAddress(address common.Address, host string) id.UserID {
	return id.NewUserID(strings.ToLower(address.Hex()), host)
}

func AddressFromUserID(userID id.UserID) common.Address {
	return common.HexToAddress(userID.Localpart())
}

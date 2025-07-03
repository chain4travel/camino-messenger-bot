// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package encoding

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

type Storage interface {
	SessionHandler
	GetBotPubKey(ctx context.Context, session Session, botAddr common.Address) (*ecdsa.PublicKey, error)
	SetBotPubKey(ctx context.Context, session Session, botAddr common.Address, pubKey *ecdsa.PublicKey) error
}

type SessionHandler interface {
	NewSession(ctx context.Context) (Session, error)
	Commit(session Session) error
	Abort(session Session)
}

type Session interface {
	Commit() error
	Abort() error
}

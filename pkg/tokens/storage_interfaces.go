// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tokenstorage

import (
	"context"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/pkg/database/sqlite"
	"github.com/jmoiron/sqlx"
)

type TokenRecord struct {
	TokenID   string        `db:"token_id"`
	TxID      string        `db:"tx_id"`
	MintID    *typesv1.UUID `db:"mint_id"`
	CreatedAt []byte        `db:"created_at"`
	ExpiresAt []byte        `db:"expires_at"`
	Bought    bool          `db:"bought"`
	Expired   bool          `db:"expired"`
}

type Session interface {
	Commit() error
	Abort() error
}

type TokenRecordsStorage interface {
	SaveTokenRecord(ctx context.Context, session Session, record *TokenRecord) error
	GetTokenRecord(ctx context.Context, session Session, tokenID string) (*TokenRecord, error)
	UpdateTokenRecord(ctx context.Context, session Session, record *TokenRecord) error
	GetActiveTokenRecords(ctx context.Context, session Session) ([]*TokenRecord, error)
}

type storage struct {
	base *sqlite.DB
	tokenRecordsStatements
}

type Storage interface {
	TokenRecordsStorage
	Close() error
	NewSession(ctx context.Context) (sqlite.Session, error)
	Commit(session sqlite.Session) error
	Abort(session sqlite.Session)
	DB() *sqlx.DB
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/encoding"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
)

const pubKeysTableName = "pub_keys"

func (s *storage) GetBotPubKey(ctx context.Context, session encoding.Session, address common.Address) (*ecdsa.PublicKey, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	pubKeyBytes := make([]byte, 0, 33) // 33 bytes for a compressed public key
	if err := tx.StmtxContext(ctx, s.getPubKey).GetContext(ctx, &pubKeyBytes, address.Bytes()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute get public key statement: %w", err)
	}

	pubKey, err := crypto.DecompressPubkey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress public key bytes: %w", err)
	}

	return pubKey, nil
}

func (s *storage) SetBotPubKey(ctx context.Context, session encoding.Session, address common.Address, pubKey *ecdsa.PublicKey) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.StmtxContext(ctx, s.setPubKey).ExecContext(ctx, address.Bytes(), crypto.CompressPubkey(pubKey))
	if err != nil {
		return fmt.Errorf("failed to execute set public key statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

type pubKeysStatements struct {
	getPubKey, setPubKey *sqlx.Stmt
}

func (s *storage) preparePubKeysStmts(ctx context.Context) error {
	getPubKey, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT pub_key FROM %s
		WHERE address = ?
	`, pubKeysTableName))
	if err != nil {
		return fmt.Errorf("failed to prepare get pub key statement: %w", err)
	}
	s.getPubKey = getPubKey

	setPubKey, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			address, pub_key
		) VALUES (?, ?)
		ON CONFLICT(address)
		DO UPDATE SET
			pub_key = excluded.pub_key
	`, pubKeysTableName))
	if err != nil {
		return fmt.Errorf("failed to prepare set pub key statement: %w", err)
	}
	s.setPubKey = setPubKey

	return nil
}

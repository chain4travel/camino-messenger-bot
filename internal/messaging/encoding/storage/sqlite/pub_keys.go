// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encoding"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/database/sqlite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
)

const pubKeysTableName = "pub_keys"

func (s *storage) GetBotPubKey(ctx context.Context, session encoding.Session, address common.Address) (*ecdsa.PublicKey, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, err
	}

	pubKeyBytes := make([]byte, 0, 33) // 33 bytes for a compressed public key
	if err := tx.StmtxContext(ctx, s.getPubKey).GetContext(ctx, &pubKeyBytes, address.Bytes()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		s.base.Logger.Error(err)
		return nil, err
	}

	pubKey, err := crypto.DecompressPubkey(pubKeyBytes)
	if err != nil {
		s.base.Logger.Error(err)
		return nil, fmt.Errorf("failed to decompress public key for address %s: %w", address.Hex(), err)
	}

	return pubKey, nil
}

func (s *storage) SetBotPubKey(ctx context.Context, session encoding.Session, address common.Address, pubKey *ecdsa.PublicKey) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.StmtxContext(ctx, s.setPubKey).ExecContext(ctx, address.Bytes(), crypto.CompressPubkey(pubKey))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
		return err
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to set public key for address %s, expected 1 row affected, got %d", address.Hex(), rowsAffected)
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
		s.base.Logger.Error(err)
		return err
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
		s.base.Logger.Error(err)
		return err
	}
	s.setPubKey = setPubKey

	return nil
}

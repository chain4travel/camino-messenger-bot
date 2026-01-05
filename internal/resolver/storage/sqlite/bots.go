// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/resolver"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/database/sqlite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const botsTable = "bots"

var _ resolver.Storage = (*storage)(nil)

func (s *storage) GetFirstBotWithStatus(ctx context.Context, session resolver.Session, cmAccount common.Address, status resolver.BotStatus) (common.Address, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return common.Address{}, err
	}

	botAddr := common.Address{}
	if err := tx.StmtxContext(ctx, s.getBot).GetContext(ctx, &botAddr, cmAccount, int(status)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return common.Address{}, resolver.ErrNotFound
		}
		s.base.Logger.Error(err)
		return common.Address{}, err
	}
	return botAddr, nil
}

func (s *storage) SetBotStatus(ctx context.Context, session resolver.Session, botAddress common.Address, status resolver.BotStatus) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	result, err := tx.StmtxContext(ctx, s.setBotStatus).ExecContext(ctx, int(status), botAddress)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.base.Logger.Error(err)
		return err
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to set bot status: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) SetBots(ctx context.Context, session resolver.Session, cmAccount common.Address, bots []common.Address) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	_, err = tx.StmtxContext(ctx, s.deleteBotsByCMAccount).ExecContext(ctx, cmAccount)
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}

	for _, botAddr := range bots {
		result, err := tx.StmtxContext(ctx, s.insertBot).ExecContext(ctx, cmAccount, botAddr)
		if err != nil {
			s.base.Logger.Error(err)
			return err
		}
		if rowsAffected, err := result.RowsAffected(); err != nil {
			s.base.Logger.Error(err)
			return err
		} else if rowsAffected != 1 {
			return fmt.Errorf("failed to insert bot: expected to affect 1 row, but affected %d", rowsAffected)
		}
	}
	return nil
}

type botsStatements struct {
	insertBot             *sqlx.Stmt
	deleteBotsByCMAccount *sqlx.Stmt
	setBotStatus          *sqlx.Stmt
	getBot                *sqlx.Stmt
}

func (s *storage) prepareBotsStmts(ctx context.Context) error {
	insertBot, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (cm_account, bot, status)
			VALUES (?, ?, %d)
	`, botsTable, resolver.BotStatusUnknown))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.insertBot = insertBot

	deleteBotsByCMAccount, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE cm_account = ?
	`, botsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.deleteBotsByCMAccount = deleteBotsByCMAccount

	setBotStatus, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET status = ?
		WHERE bot = ?
	`, botsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.setBotStatus = setBotStatus

	getBot, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT bot FROM %s
		WHERE cm_account = ? AND status = ?
		LIMIT 1
	`, botsTable))
	if err != nil {
		s.base.Logger.Error(err)
		return err
	}
	s.getBot = getBot

	return nil
}

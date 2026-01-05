// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package resolver

import (
	"context"
	"errors"
	"fmt"

	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

var (
	_ Resolver = (*resolver)(nil)

	ErrNotFound = fmt.Errorf("no bot found for the given CM account and status")
)

type BotStatus uint8

const (
	BotStatusUnknown     BotStatus = 0
	BotStatusReachable   BotStatus = 1
	BotStatusUnreachable BotStatus = 2
)

type Resolver interface {
	GetBotAddress(ctx context.Context, recipientCMAccount common.Address) (common.Address, error)
	SetBotStatus(ctx context.Context, botAddress common.Address, status BotStatus) error
}

type Storage interface {
	SessionHandler
	GetFirstBotWithStatus(ctx context.Context, session Session, cmAccount common.Address, status BotStatus) (common.Address, error)
	SetBotStatus(ctx context.Context, session Session, botAddress common.Address, status BotStatus) error
	SetBots(ctx context.Context, session Session, cmAccount common.Address, bots []common.Address) error
}

type SessionHandler interface {
	NewSession(context.Context) (Session, error)
	Commit(Session) error
	Abort(Session)
}

type Session interface {
	Commit() error
	Abort() error
}
type resolver struct {
	logger     *zap.SugaredLogger
	cmAccounts cmaccounts.Service
	storage    Storage
}

func NewResolver(logger *zap.SugaredLogger, cmAccounts cmaccounts.Service, storage Storage) Resolver {
	return &resolver{
		logger:     logger,
		cmAccounts: cmAccounts,
		storage:    storage,
	}
}

func (r *resolver) GetBotAddress(ctx context.Context, recipientCMAccount common.Address) (common.Address, error) {
	session, err := r.storage.NewSession(ctx)
	if err != nil {
		return common.Address{}, err
	}
	defer r.storage.Abort(session)

	reachableBot, err := r.storage.GetFirstBotWithStatus(ctx, session, recipientCMAccount, BotStatusReachable)
	switch {
	case err == nil:
		return reachableBot, nil
	case !errors.Is(err, ErrNotFound):
		return common.Address{}, fmt.Errorf("failed to get first reachable bot from db: %w", err)
	}

	r.logger.Infof("no reachable bot found for recipient CM account %s in db, checking for unknown status bots", recipientCMAccount.Hex())

	reachableBot, err = r.storage.GetFirstBotWithStatus(ctx, session, recipientCMAccount, BotStatusUnknown)
	switch {
	case err == nil:
		return reachableBot, nil
	case !errors.Is(err, ErrNotFound):
		return common.Address{}, fmt.Errorf("failed to get first bot with unknown reachability status from db: %w", err)
	}

	r.logger.Infof("no unknown status bot found for recipient CM account %s in db, fetching bots from blockchain", recipientCMAccount.Hex())

	recipientBots, err := r.cmAccounts.GetAllChequeOperators(ctx, recipientCMAccount)
	if err != nil {
		return common.Address{}, err
	}

	if err := r.storage.SetBots(ctx, session, recipientCMAccount, recipientBots); err != nil {
		return common.Address{}, fmt.Errorf("failed to set bots in db: %w", err)
	}

	if err := r.storage.Commit(session); err != nil {
		return common.Address{}, fmt.Errorf("failed to commit session: %w", err)
	}

	if len(recipientBots) == 0 {
		return common.Address{}, fmt.Errorf("no bots found for recipient CM account %s: %w", recipientCMAccount.Hex(), ErrNotFound)
	}

	r.logger.Infof("using first bot %s with unknown reachability status for recipient CM account %s", recipientBots[0].Hex(), recipientCMAccount.Hex())

	return recipientBots[0], nil
}

func (r *resolver) SetBotStatus(ctx context.Context, botAddress common.Address, status BotStatus) error {
	session, err := r.storage.NewSession(ctx)
	if err != nil {
		return err
	}
	defer r.storage.Abort(session)

	if err := r.storage.SetBotStatus(ctx, session, botAddress, status); err != nil {
		return fmt.Errorf("failed to set bot status in db: %w", err)
	}

	if err := r.storage.Commit(session); err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}

	return nil
}

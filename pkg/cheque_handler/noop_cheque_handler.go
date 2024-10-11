/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package chequehandler

import (
	"context"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
)

var _ ChequeHandler = (*NoopChequeHandler)(nil)

type NoopChequeHandler struct{}

func (NoopChequeHandler) IssueCheque(context.Context, common.Address, common.Address, common.Address, *big.Int) (*cheques.SignedCheque, error) {
	return &cheques.SignedCheque{}, nil
}

func (NoopChequeHandler) IsAllowedToIssueCheque(context.Context, common.Address) (bool, error) {
	return true, nil
}

func (NoopChequeHandler) CashIn(context.Context) error {
	return nil
}

func (NoopChequeHandler) CheckCashInStatus(context.Context) error {
	return nil
}

func (NoopChequeHandler) VerifyCheque(context.Context, *cheques.SignedCheque, common.Address, *big.Int) error {
	return nil
}

/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
)

var _ ChequeHandler = (*NoopChequeHandler)(nil)

type NoopChequeHandler struct{}

func (NoopChequeHandler) IssueCheque(_ context.Context, _ common.Address, _ common.Address, _ common.Address, _ common.Address, _ *big.Int) (*cheques.SignedCheque, error) {
	return &cheques.SignedCheque{}, nil
}

func (NoopChequeHandler) GetServiceFee(_ context.Context, _ common.Address, _ MessageType) (*big.Int, error) {
	return nil, nil
}

func (NoopChequeHandler) IsBotAllowed(_ context.Context, _ common.Address) (bool, error) {
	return true, nil
}

func (NoopChequeHandler) IsEmptyCheque(_ *cheques.SignedCheque) bool {
	return false
}

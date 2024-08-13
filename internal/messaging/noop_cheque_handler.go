/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var _ ChequeHandler = (*NoopChequeHandler)(nil)

type NoopChequeHandler struct{}

func (NoopChequeHandler) HandleResponse(context.Context, MessageType, *RequestContent, *ResponseContent) {
}

func (NoopChequeHandler) HandleRequest(context.Context, MessageType, *RequestContent) error {
	return nil
}

func (NoopChequeHandler) issueCheque(ctx context.Context, fromCMAccount string, toCMAccount string, toBot string, amount *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopChequeHandler) signCheque(ctx context.Context, cheque Cheque) ([]byte, error) {
	return nil, nil
}
func (NoopChequeHandler) getLastCashIn(ctx context.Context, fromBot common.Address, toBot common.Address) (*LastCashIn, error) {
	return nil, nil
}
func (NoopChequeHandler) verifyCheque(ctx context.Context, cheque Cheque, signature []byte) (*ChequeVerifiedEvent, error) {
	return nil, nil
}
func (NoopChequeHandler) getServiceFeeByName(serviceName string) (*big.Int, error) {
	return nil, nil
}

/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct{}

func (NoopResponseHandler) HandleResponse(context.Context, MessageType, *RequestContent, *ResponseContent) {
}

func (NoopResponseHandler) HandleRequest(context.Context, MessageType, *RequestContent) error {
	return nil
}
func (NoopResponseHandler) getChequeVerifiedEvent(txHash common.Hash) (*ChequeVerifiedEvent, error) {
	return nil, nil
}

func (NoopResponseHandler) getServiceFeeByName(serviceName string, CMAccountAddress common.Address) (*big.Int, error) {
	return nil, nil
}
func (NoopResponseHandler) serviceNameToHash(serviceName string) string {
	return ""
}
func (NoopResponseHandler) getServiceFee(serviceHash string) (big.Int, error) {
	return *big.NewInt(0), nil
}

func (NoopResponseHandler) isBotAllowed() (bool, error) {
	return false, nil
}

func (NoopResponseHandler) getAddressFromECDSAPrivateKey(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	return common.Address{}, nil
}

func (NoopResponseHandler) getLastCashIn(ctx context.Context, fromBot common.Address, toBot common.Address) (*LastCashIn, error) {
	return nil, nil
}

func (NoopResponseHandler) issueCheque(ctx context.Context, fromCMAccount common.Address, toCMAccount common.Address, toBot common.Address, amount *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopResponseHandler) signCheque(ctx context.Context, cheque Cheque) ([]byte, error) {
	return nil, nil
}

func (NoopResponseHandler) getAllBotAddressesFromCMAccountAddress(common.Address) ([]string, error) {
	return nil, nil
}

func (NoopResponseHandler) getSingleBotFromCMAccountAddress(common.Address) (string, error) {
	return "", nil
}

func (NoopResponseHandler) isMyCMAccount(common.Address) bool {
	return false
}

func (NoopResponseHandler) getMyCMAccountAddress() string {
	return ""
}

func (NoopResponseHandler) getMatrixHost() string {
	return ""
}

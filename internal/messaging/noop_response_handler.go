/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct{}

func (NoopResponseHandler) HandleResponse(context.Context, MessageType, *RequestContent, *ResponseContent) {
}

func (NoopResponseHandler) HandleRequest(context.Context, MessageType, *RequestContent) error {
	return nil
}

func (NoopResponseHandler) getAllBotAddressesFromCMAccountAddress(common.Address) ([]string, error) {
	return nil, nil
}

func (NoopResponseHandler) getFirstBotFromCMAccountAddress(common.Address) (string, error) {
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

func (NoopResponseHandler) isBotInCMAccount(string, common.Address) (bool, error) {
	return false, nil
}

func (NoopResponseHandler) addToMap(_ common.Address, _ string) {
}

func (NoopResponseHandler) getCmAccount(_ string) (common.Address, bool) {
	return common.Address{}, false
}

func (NoopResponseHandler) removeFromMap(cmaccount common.Address) {
}

func (NoopResponseHandler) getBotFromMap(cmaccount common.Address) (bool, string) {
	return false, ""
}

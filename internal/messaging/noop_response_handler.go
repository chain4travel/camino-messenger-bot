/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"maunium.net/go/mautrix/id"
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

func (NoopResponseHandler) getMyCMAccountAddress() common.Address {
	return common.Address{}
}

func (NoopResponseHandler) getMatrixHost() string {
	return ""
}

func (NoopResponseHandler) isBotInCMAccount(common.Address, common.Address) (bool, error) {
	return false, nil
}

func (NoopResponseHandler) addToMap(_ common.Address, _ id.UserID) {
}

func (NoopResponseHandler) getCmAccount(_ id.UserID) (common.Address, bool) {
	return common.Address{}, false
}

func (NoopResponseHandler) getBotFromMap(_ common.Address) (bool, id.UserID) {
	return false, ""
}

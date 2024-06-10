/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
)

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct{}

func (NoopResponseHandler) HandleResponse(context.Context, MessageType, *RequestContent, *ResponseContent) {
}

func (NoopResponseHandler) HandleRequest(ctx context.Context, msgType MessageType, request *RequestContent) error {
	return nil
}

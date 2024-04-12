/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import "context"

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct {
}

func (n NoopResponseHandler) HandleResponse(context.Context, MessageType, *ResponseContent) error {
	return nil
}

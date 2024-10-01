/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct{}

func (NoopResponseHandler) HandleResponse(context.Context, types.MessageType, protoreflect.ProtoMessage, protoreflect.ProtoMessage) {
}

func (NoopResponseHandler) HandleRequest(context.Context, types.MessageType, protoreflect.ProtoMessage) error {
	return nil
}

func (NoopResponseHandler) AddErrorToResponseHeader(types.MessageType, protoreflect.ProtoMessage, string) {
}

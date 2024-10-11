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

func (NoopResponseHandler) ProcessResponseMessage(context.Context, *types.Message, *types.Message) {}

func (NoopResponseHandler) PrepareResponseMessage(context.Context, *types.Message, *types.Message) {}

func (NoopResponseHandler) PrepareRequest(types.MessageType, protoreflect.ProtoMessage) error {
	return nil
}

func (NoopResponseHandler) AddErrorToResponseHeader(protoreflect.ProtoMessage, string) {}

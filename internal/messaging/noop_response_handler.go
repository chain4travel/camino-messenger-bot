// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ ResponseHandler = (*NoopResponseHandler)(nil)

type NoopResponseHandler struct{}

func (NoopResponseHandler) ProcessResponseMessage(context.Context, *types.Message, *types.Message) {}

func (NoopResponseHandler) PrepareResponseMessage(context.Context, *types.Message, *types.Message) {}

func (NoopResponseHandler) PrepareRequest(protoreflect.ProtoMessage) error {
	return nil
}

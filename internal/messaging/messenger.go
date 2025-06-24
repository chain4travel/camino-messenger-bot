// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"maunium.net/go/mautrix/id"
)

type APIMessageResponse struct {
	Message types.Message
	Err     error
}
type Messenger interface {
	// Initializes messenger and starts receiving messages.
	Start(ctx context.Context) (chan error, error)

	// Stop receiving messages.
	Stop() error

	// Should only be called after Start() was called.
	SendMessage(ctx context.Context, m *types.Message, sendTo id.UserID) error

	// Channel where incoming messages are written
	Inbound() chan types.Message
}

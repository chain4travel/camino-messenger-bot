/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/id"
)

type APIMessageResponse struct {
	Message Message
	Err     error
}
type Messenger interface {
	metadata.Checkpoint
	StartReceiver() (id.UserID, error)                                                  // start receiving messages. Returns the user id
	StopReceiver() error                                                                // stop receiving messages
	SendAsync(ctx context.Context, m Message, content [][]byte, sendTo id.UserID) error // asynchronous call (fire and forget)
	Inbound() chan Message                                                              // channel where incoming messages are written
}

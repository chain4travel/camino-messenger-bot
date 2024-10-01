/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/id"
)

type APIMessageResponse struct {
	Message messages.Message
	Err     error
}
type Messenger interface {
	metadata.Checkpoint

	// start receiving messages. Returns the user id
	StartReceiver() (id.UserID, error)

	// stop receiving messages
	StopReceiver() error

	// asynchronous call (fire and forget)
	SendAsync(ctx context.Context, m messages.Message, content [][]byte, sendTo id.UserID) error

	// channel where incoming messages are written
	Inbound() chan messages.Message
}

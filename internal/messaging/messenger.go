package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
)

type APIMessageResponse struct {
	Message Message
	Err     error
}
type Messenger interface {
	metadata.Checkpoint
	StartReceiver(botMode uint) (string, error) // start receiving messages. Returns the user id
	StopReceiver() error                        // stop receiving messages
	SendAsync(context.Context, Message) error   // asynchronous call (fire and forget)
	Inbound() chan Message                      // channel where incoming messages are written
}

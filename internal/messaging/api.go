package messaging

import "context"

type APIMessageResponse struct {
	Message Message
	Err     error
}
type Messenger interface {
	StartReceiver() error                           // start receiving messages
	StopReceiver() error                            // stop receiving messages
	SendAsync(ctx context.Context, m Message) error // asynchronous call (fire and forget)
	Inbound() chan Message                          // channel where incoming messages are written
}

package matrix

import (
	"camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
)

type CaminoMatrixMessage struct {
	event.MessageEventContent
	Metadata metadata.Metadata `json:"Metadata"`
}

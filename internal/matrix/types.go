package matrix

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
)

type CaminoMatrixMessage struct {
	event.MessageEventContent
	Content  messaging.MessageContent `json:"content"`
	Metadata metadata.Metadata        `json:"Metadata"`
}

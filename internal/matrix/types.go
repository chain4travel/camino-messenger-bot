package matrix

import (
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
)

type CaminoMatrixMessage struct {
	event.MessageEventContent
	Metadata metadata.Metadata `json:"Metadata"`
}

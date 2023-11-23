package matrix

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type CaminoMatrixMessage struct {
	event.MessageEventContent
	Content           messaging.MessageContent `json:"content"`
	CompressedContent []byte                   `json:"compressed_content"`
	Metadata          metadata.Metadata        `json:"metadata"`
}

type caminoMsgEventPayload struct {
	roomID          id.RoomID
	eventType       event.Type
	caminoMatrixMsg CaminoMatrixMessage
}

type ByChunkIndex []caminoMsgEventPayload

func (b ByChunkIndex) Len() int { return len(b) }
func (b ByChunkIndex) Less(i, j int) bool {
	return b[i].caminoMatrixMsg.Metadata.ChunkIndex < b[j].caminoMatrixMsg.Metadata.ChunkIndex
}
func (b ByChunkIndex) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

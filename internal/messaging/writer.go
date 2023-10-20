package messaging

import "camino-messenger-provider/internal/matrix"

type Writer interface {
	Write(roomID string, message matrix.TimelineEventContent) error
}

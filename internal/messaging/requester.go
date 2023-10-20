package messaging

import "camino-messenger-provider/internal/matrix"

var (
	_ Writer = (*requester)(nil)
)

type requester struct {
	matrixClient matrix.Client
}

func (r *requester) Write(roomID string, message matrix.TimelineEventContent) error {

	// issue and attach cheques
	message.Cheques = nil //TODO
	// forward message to matrix
	return r.matrixClient.Send(roomID, message)
}

package messaging

import "camino-messenger-provider/internal/matrix"

var (
	_ Writer = (*responder)(nil)
)

type responder struct {
	matrixClient matrix.Client
}

func (r *responder) Write(roomID string, message matrix.TimelineEventContent) error {
	//TODO implement me
	panic("implement me")

	return r.matrixClient.Send(roomID, message)
}

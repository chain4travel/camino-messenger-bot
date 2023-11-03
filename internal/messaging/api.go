package messaging

type Metadata struct {
	Sender  string
	Cheques []map[string]interface{}
}
type Message struct {
	RequestID string
	Type      MessageType
	Body      string
	Metadata  Metadata
}

type APIMessageResponse struct {
	Message Message
	Err     error
}
type Messenger interface {
	StartReceiver() error      // start receiving messages
	StopReceiver() error       // stop receiving messages
	SendAsync(m Message) error // asynchronous call (fire and forget)
	Inbound() chan Message     // channel where incoming messages are written
}

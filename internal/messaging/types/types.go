package types

import (
	"errors"
	"strings"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"maunium.net/go/mautrix/id"
)

var ErrUnknownMessageType = errors.New("unknown message type")

type MessageCategory byte

const (
	Request MessageCategory = iota
	Response
	Unknown
)

// Always has to be in the format <ServiceName>.<Request/Response>
type MessageType string

func (m MessageType) ToServiceName() string {
	msgStr := string(m)
	if idx := strings.LastIndex(msgStr, "."); idx != -1 {
		return msgStr[:idx]
	}
	return ""
}

func (m MessageType) Category() MessageCategory {
	switch {
	case strings.HasSuffix(string(m), ".Request"):
		return Request
	case strings.HasSuffix(string(m), ".Response"):
		return Response
	}
	return Unknown
}

func ServiceNameToRequestMessageType(serviceName string) MessageType {
	return MessageType(serviceName + ".Request")
}

// Message is the message format used for communication between the messenger and the service
type Message struct {
	Type     MessageType               `json:"msgtype"`
	Content  protoreflect.ProtoMessage `json:"content"`
	Metadata metadata.Metadata         `json:"metadata"`
	Sender   id.UserID
}

func (m *Message) MarshalContent() ([]byte, error) {
	return proto.Marshal(m.Content)
}

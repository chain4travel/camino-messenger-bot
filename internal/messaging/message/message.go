// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"errors"
	"strings"

	"github.com/chain4travel/camino-messenger-bot/v12/pkg/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrUnknownMessageType = errors.New("unknown message type")

type Category byte

const (
	Unknown Category = iota
	Request
	Response
)

// Always has to be in the format <ServiceName>.<Request/Response>
type Type string

func (m Type) ToServiceName() string {
	msgStr := string(m)
	if idx := strings.LastIndex(msgStr, "."); idx != -1 {
		return msgStr[:idx]
	}
	return ""
}

func (m Type) Category() Category {
	switch {
	case strings.HasSuffix(string(m), ".Request"):
		return Request
	case strings.HasSuffix(string(m), ".Response"):
		return Response
	}
	return Unknown
}

func ServiceNameToRequestMessageType(serviceName string) Type {
	return Type(serviceName + ".Request")
}

type Message struct {
	Type       Type
	Content    protoreflect.ProtoMessage
	RequestID  string
	Timestamps metadata.Timestamps
}

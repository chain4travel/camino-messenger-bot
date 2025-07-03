// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package types

import (
	"errors"
	"strings"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrUnknownMessageType = errors.New("unknown message type")

type MessageCategory byte

const (
	Unknown MessageCategory = iota
	Request
	Response
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

type Message struct {
	Type       MessageType
	Content    protoreflect.ProtoMessage
	RequestID  string
	Timestamps metadata.Timestamps
}

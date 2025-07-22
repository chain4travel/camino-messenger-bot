// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"encoding/json"
	"fmt"
	"time"
)

type Checkpoint string

const (
	CheckpointGRPCRequestReceived                  = "grpc_request_received"
	CheckpointP2PRequestMessageSentToServer        = "p2p_request_message_sent_to_server"
	CheckpointP2PRequestMessageReceivedFromServer  = "p2p_request_message_received_from_server"
	CheckpointP2PRequestMessageSentToPP            = "p2p_request_message_sent_to_pp"
	CheckpointP2PResponseMessageReceivedFromPP     = "p2p_response_message_received_from_pp"
	CheckpointP2PResponseMessageSentToServer       = "p2p_response_message_sent_to_server"
	CheckpointP2PResponseMessageReceivedFromServer = "p2p_response_message_received_from_server"
	CheckpointGRPCResponseSent                     = "grpc_response_sent"
)

type Timestamps map[string]int64

func TimestampsFromString(s string) (Timestamps, error) {
	var timestamps Timestamps
	if err := json.Unmarshal([]byte(s), &timestamps); err != nil {
		return nil, fmt.Errorf("error unmarshalling timestamps: %w", err)
	}
	return timestamps, nil
}

func (t Timestamps) Stamp(checkpoint Checkpoint) {
	// order-checkpoint -> timestamp milliseconds
	t[fmt.Sprintf("%d-%s", len(t), checkpoint)] = time.Now().UnixMilli()
}

func (t Timestamps) MarshalToString() (string, error) {
	timestampsJSON, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("error marshalling timestamps: %w", err)
	}
	return string(timestampsJSON), nil
}

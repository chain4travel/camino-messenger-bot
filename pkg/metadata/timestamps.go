// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"encoding/json"
	"fmt"
	"time"
)

type Checkpoint int

const (
	CheckpointP2PRequestReceived Checkpoint = iota
	CheckpointP2PRequestMessageSentToServer
	CheckpointP2PRequestMessageReceivedFromServer
	CheckpointP2PRequestMessageSentToPP
	CheckpointP2PResponseMessageReceivedFromPP
	CheckpointP2PResponseMessageSentToServer
	CheckpointP2PResponseMessageReceivedFromServer
	CheckpointP2PResponseSent
)

func (c Checkpoint) String() string {
	switch c {
	case CheckpointP2PRequestReceived:
		return "p2p_request_received"
	case CheckpointP2PRequestMessageSentToServer:
		return "p2p_request_message_sent_to_server"
	case CheckpointP2PRequestMessageReceivedFromServer:
		return "p2p_request_message_received_from_server"
	case CheckpointP2PRequestMessageSentToPP:
		return "p2p_request_message_sent_to_pp"
	case CheckpointP2PResponseMessageReceivedFromPP:
		return "p2p_response_message_received_from_pp"
	case CheckpointP2PResponseMessageSentToServer:
		return "p2p_response_message_sent_to_server"
	case CheckpointP2PResponseMessageReceivedFromServer:
		return "p2p_response_message_received_from_server"
	case CheckpointP2PResponseSent:
		return "p2p_response_sent"
	default:
		return fmt.Sprintf("unknown_checkpoint_%d", c)
	}
}

type Timestamps map[string]int64

func TimestampsFromString(s string) (Timestamps, error) {
	var timestamps Timestamps
	if err := json.Unmarshal([]byte(s), &timestamps); err != nil {
		return nil, fmt.Errorf("error unmarshalling timestamps: %w", err)
	}
	return timestamps, nil
}

func (t Timestamps) Stamp(checkpoint Checkpoint) {
	t.StampOn(checkpoint, time.Now())
}

func (t Timestamps) StampOn(checkpoint Checkpoint, time time.Time) {
	// order-checkpoint -> timestamp
	t[fmt.Sprintf("%d-%d", len(t), checkpoint)] = time.UnixMilli()
}

func (t Timestamps) MarshalToString() (string, error) {
	timestampsJSON, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("error marshalling timestamps: %w", err)
	}
	return string(timestampsJSON), nil
}

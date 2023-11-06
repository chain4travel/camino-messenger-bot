package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

type Metadata struct {
	RequestID  string                   `json:"request_id"`
	Sender     string                   `json:"sender"`
	RoomID     string                   `json:"room_id"`
	Cheques    []map[string]interface{} `json:"cheques"`
	Timestamps map[string]int64         `json:"timestamps"`
}

func (m *Metadata) ExtractMetadata(ctx context.Context) error {
	mdPairs, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return fmt.Errorf("metadata not found in context")
	}
	if requestID, found := mdPairs["request_id"]; found {
		m.RequestID = requestID[0]
	}

	if sender, found := mdPairs["sender"]; found {
		m.Sender = sender[0]
	}

	if roomID, found := mdPairs["room_id"]; found {
		m.RoomID = roomID[0]
	}

	if cheques, found := mdPairs["cheques"]; found {
		chequesJSON := strings.Join(cheques, "")
		if err := json.Unmarshal([]byte(chequesJSON), &m.Cheques); err != nil {
			return fmt.Errorf("error unmarshalling cheques: %v", err)
		}
	}

	if timestamps, found := mdPairs["timestamps"]; found {
		timestampsJSON := strings.Join(timestamps, "")
		if err := json.Unmarshal([]byte(timestampsJSON), &m.Timestamps); err != nil {
			return fmt.Errorf("error unmarshalling timestamps: %v", err)
		}
	}
	return nil
}

func (m *Metadata) ToGrpcMD() metadata.MD {
	md := metadata.New(map[string]string{
		"sender":  m.Sender,
		"room_id": m.RoomID,
		"timestamps": func() string {
			timestampsJSON, _ := json.Marshal(m.Timestamps)
			return string(timestampsJSON)
		}(),
	})
	return md
}
func (m *Metadata) Stamp(checkpoint string) {
	if m.Timestamps == nil {
		m.Timestamps = make(map[string]int64)
	}
	m.Timestamps[checkpoint] = time.Now().Unix()
}

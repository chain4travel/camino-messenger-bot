// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"google.golang.org/grpc/metadata"
)

type Metadata struct {
	RequestID          string                 `json:"request_id"`
	SenderCMAccount    string                 `json:"sender"`
	RecipientCMAccount string                 `json:"recipient"`
	Cheques            []cheques.SignedCheque `json:"cheques"`
	Timestamps         map[string]int64       `json:"timestamps"` // map of checkpoints to timestamps in unix milliseconds
	NumberOfChunks     uint64                 `json:"number_of_chunks"`
	ChunkIndex         uint64                 `json:"chunk_index"`

	// Deprecated: this metadata serves only as a temp solution and should be removed and addressed on the protocol level
	ProviderOperator string `json:"provider_operator"`
}

func (m *Metadata) ExtractMetadata(ctx context.Context) error {
	mdPairs, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		mdPairs, ok = metadata.FromOutgoingContext(ctx)
		if !ok {
			return fmt.Errorf("metadata not found in context")
		}
	}
	return m.FromGrpcMD(mdPairs)
}

func (m *Metadata) FromGrpcMD(mdPairs metadata.MD) error {
	if requestID, found := mdPairs["request_id"]; found && len(requestID[0]) > 0 {
		m.RequestID = requestID[0]
	}

	if sender, found := mdPairs["sender"]; found && len(sender[0]) > 0 {
		m.SenderCMAccount = sender[0]
	}

	if recipient, found := mdPairs["recipient"]; found && len(recipient[0]) > 0 {
		m.RecipientCMAccount = recipient[0]
	}

	if cheques, found := mdPairs["cheques"]; found && len(cheques[0]) > 0 {
		chequesJSON := strings.Join(cheques, "")
		if err := json.Unmarshal([]byte(chequesJSON), &m.Cheques); err != nil {
			return fmt.Errorf("error unmarshalling cheques: %w", err)
		}
	}

	if timestamps, found := mdPairs["timestamps"]; found && len(timestamps[0]) > 0 {
		timestampsJSON := strings.Join(timestamps, "")
		if err := json.Unmarshal([]byte(timestampsJSON), &m.Timestamps); err != nil {
			return fmt.Errorf("error unmarshalling timestamps: %w", err)
		}
	}
	if providerOperator, found := mdPairs["provider_operator"]; found && len(providerOperator[0]) > 0 {
		m.ProviderOperator = providerOperator[0]
	}
	return nil
}

func (m *Metadata) ToGrpcMD() metadata.MD {
	md := metadata.New(map[string]string{
		"request_id": m.RequestID,
		"sender":     m.SenderCMAccount,
		"recipient":  m.RecipientCMAccount,
		"timestamps": func() string {
			timestampsJSON, _ := json.Marshal(m.Timestamps)
			return string(timestampsJSON)
		}(),
		"cheques": func() string {
			chequesJSON, _ := json.Marshal(m.Cheques)
			return string(chequesJSON)
		}(),
		"provider_operator": m.ProviderOperator,
	})
	return md
}

func (m *Metadata) Stamp(checkpoint string) {
	if m.Timestamps == nil {
		m.Timestamps = make(map[string]int64)
	}
	idx := len(m.Timestamps) // for analysis' sake, we want to know the order of the checkpoints
	m.Timestamps[fmt.Sprintf("%d-%s", idx, checkpoint)] = time.Now().UnixMilli()
}

func (m *Metadata) StampOn(checkpoint string, t int64) {
	if m.Timestamps == nil {
		m.Timestamps = make(map[string]int64)
	}
	idx := len(m.Timestamps) // for analysis' sake, we want to know the order of the checkpoints
	m.Timestamps[fmt.Sprintf("%d-%s", idx, checkpoint)] = t
}

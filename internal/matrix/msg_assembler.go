/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"sync"

	"go.uber.org/zap"
)

type MessageAssembler interface {
	AssembleMessage(msg CaminoMatrixMessage) (CaminoMatrixMessage, error, bool) // returns assembled message and true if message is complete. Otherwise, it returns an empty message and false
}

type messageAssembler struct {
	logger          *zap.SugaredLogger
	partialMessages map[string][]CaminoMatrixMessage
	mu              sync.RWMutex
}

func NewMessageAssembler(logger *zap.SugaredLogger) MessageAssembler {
	return &messageAssembler{logger: logger, partialMessages: make(map[string][]CaminoMatrixMessage)}
}
func (a *messageAssembler) AssembleMessage(msg CaminoMatrixMessage) (CaminoMatrixMessage, error, bool) {
	if msg.Metadata.NumberOfChunks == 1 {
		decompressedCaminoMsg, err := assembleAndDecompressCaminoMatrixMessages([]CaminoMatrixMessage{msg})
		return decompressedCaminoMsg, err, true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	id := msg.Metadata.RequestID
	if _, ok := a.partialMessages[id]; !ok {
		a.partialMessages[id] = []CaminoMatrixMessage{}
	}

	a.partialMessages[id] = append(a.partialMessages[id], msg)
	if len(a.partialMessages[id]) == int(msg.Metadata.NumberOfChunks) {
		decompressedCaminoMsg, err := assembleAndDecompressCaminoMatrixMessages(a.partialMessages[id])
		delete(a.partialMessages, id)
		return decompressedCaminoMsg, err, true
	}
	return CaminoMatrixMessage{}, nil, false
}

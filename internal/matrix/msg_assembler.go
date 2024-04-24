/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
)

var (
	ErrDecompressFailed = errors.New("failed to decompress assembled camino matrix msg")
	ErrUnmarshalContent = errors.New("failed to unmarshal content")
)

type MessageAssembler interface {
	AssembleMessage(msg *CaminoMatrixMessage) (assembledMsg *CaminoMatrixMessage, complete bool, err error) // returns assembled message and true if message is complete. Otherwise, it returns an empty message and false
}

type messageAssembler struct {
	partialMessages map[string][]*CaminoMatrixMessage
	decompressor    compression.Decompressor
	mu              sync.RWMutex
}

func NewMessageAssembler() MessageAssembler {
	return &messageAssembler{decompressor: &compression.ZSTDDecompressor{}, partialMessages: make(map[string][]*CaminoMatrixMessage)}
}

func (a *messageAssembler) AssembleMessage(msg *CaminoMatrixMessage) (*CaminoMatrixMessage, bool, error) {
	if msg.Metadata.NumberOfChunks == 1 {
		decompressedCaminoMsg, err := a.assembleAndDecompressCaminoMatrixMessages([]*CaminoMatrixMessage{msg})
		return decompressedCaminoMsg, err == nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	id := msg.Metadata.RequestID
	if _, ok := a.partialMessages[id]; !ok {
		a.partialMessages[id] = []*CaminoMatrixMessage{}
	}

	a.partialMessages[id] = append(a.partialMessages[id], msg)
	if len(a.partialMessages[id]) == int(msg.Metadata.NumberOfChunks) {
		decompressedCaminoMsg, err := a.assembleAndDecompressCaminoMatrixMessages(a.partialMessages[id])
		delete(a.partialMessages, id)
		return decompressedCaminoMsg, err == nil, err
	}
	return nil, false, nil
}

func (a *messageAssembler) assembleAndDecompressCaminoMatrixMessages(messages []*CaminoMatrixMessage) (*CaminoMatrixMessage, error) {
	compressedPayloads := make([][]byte, 0, len(messages))

	// chunks have to be sorted
	sort.Sort(ByChunkIndex(messages))
	for _, msg := range messages {
		compressedPayloads = append(compressedPayloads, msg.CompressedContent)
	}

	// assemble chunks and decompress
	originalContent, err := a.decompressor.Decompress(assemble(compressedPayloads))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecompressFailed, err)
	}

	msg := CaminoMatrixMessage{
		MessageEventContent: messages[0].MessageEventContent,
		Metadata:            messages[0].Metadata,
	}
	err = msg.UnmarshalContent(originalContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %w %v", ErrUnmarshalContent, err, msg.MsgType)
	}

	return &msg, nil
}

func assemble(src [][]byte) []byte {
	totalLength := 0
	for _, slice := range src {
		totalLength += len(slice)
	}

	result := make([]byte, totalLength)
	index := 0
	for _, slice := range src {
		copy(result[index:], slice)
		index += len(slice)
	}
	return result
}

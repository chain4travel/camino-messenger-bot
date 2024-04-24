/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
)

var (
	_                              compression.Compressor[messaging.Message, []CaminoMatrixMessage] = (*ChunkingCompressor)(nil)
	ErrCompressionProducedNoChunks                                                                  = errors.New("compression produced no chunks")
	ErrEncodingMsg                                                                                  = errors.New("error while encoding msg for compression")
)

// ChunkingCompressor is a concrete implementation of Compressor with chunking functionality
type ChunkingCompressor struct {
	maxChunkSize int
}

// Compress implements the Compressor interface for ChunkingCompressor
func (c *ChunkingCompressor) Compress(msg messaging.Message) ([]CaminoMatrixMessage, error) {
	var matrixMessages []CaminoMatrixMessage

	// 1. Compress the message
	compressedContent, err := compress(msg)
	if err != nil {
		return matrixMessages, err
	}

	// 2. Split the compressed content into chunks
	splitCompressedContent, err := c.split(compressedContent)
	if err != nil {
		return matrixMessages, err
	}

	// 3. Create CaminoMatrixMessage objects for each chunk
	return splitCaminoMatrixMsg(msg, splitCompressedContent)
}

func (c *ChunkingCompressor) split(bytes []byte) ([][]byte, error) {
	splitCompressedContent := splitByteArray(bytes, c.maxChunkSize)

	if len(splitCompressedContent) == 0 {
		return nil, ErrCompressionProducedNoChunks // should never happen
	}
	return splitCompressedContent, nil
}

func compress(msg messaging.Message) ([]byte, error) {
	var (
		bytes []byte
		err   error
	)
	bytes, err = msg.MarshalContent()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodingMsg, err)
	}
	return compression.Compress(bytes), nil
}

func splitCaminoMatrixMsg(msg messaging.Message, splitCompressedContent [][]byte) ([]CaminoMatrixMessage, error) {
	messages := make([]CaminoMatrixMessage, 0, len(splitCompressedContent))

	// add first chunk to messages slice
	{
		caminoMatrixMsg := CaminoMatrixMessage{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            msg.Metadata,
		}
		caminoMatrixMsg.Metadata.NumberOfChunks = uint(len(splitCompressedContent))
		caminoMatrixMsg.Metadata.ChunkIndex = 0
		caminoMatrixMsg.CompressedContent = splitCompressedContent[0]
		messages = append(messages, caminoMatrixMsg)
	}

	// if multiple chunks were produced upon compression, add them to messages slice
	for i, chunk := range splitCompressedContent[1:] {
		messages = append(messages, CaminoMatrixMessage{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            metadata.Metadata{RequestID: msg.Metadata.RequestID, NumberOfChunks: uint(len(splitCompressedContent)), ChunkIndex: uint(i + 1)},
			CompressedContent:   chunk,
		})
	}

	return messages, nil
}

func splitByteArray(src []byte, maxSize int) [][]byte {
	numChunks := (len(src) + maxSize - 1) / maxSize
	result := make([][]byte, numChunks)

	start := 0
	for i := 0; i < numChunks; i++ {
		end := start + maxSize
		if end > len(src) {
			end = len(src)
		}
		result[i] = make([]byte, end-start)
		copy(result[i], src[start:end])
		start = end
	}

	return result
}

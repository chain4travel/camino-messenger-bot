/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
)

var (
	_ compression.Compressor[*types.Message, [][]byte] = (*chunkingCompressor)(nil)
	_ compression.Compressor[*types.Message, [][]byte] = (*noopCompressor)(nil)

	ErrCompressionProducedNoChunks = errors.New("compression produced no chunks")
	ErrEncodingMsg                 = errors.New("error while encoding msg for compression")
)

func NewCompressor(maxChunkSize int) compression.Compressor[*types.Message, [][]byte] {
	return &chunkingCompressor{maxChunkSize: maxChunkSize}
}

// chunkingCompressor is a concrete implementation of Compressor with chunking functionality
type chunkingCompressor struct {
	maxChunkSize int
}

// Compress implements the Compressor interface for chunkingCompressor
func (c *chunkingCompressor) Compress(msg *types.Message) ([][]byte, error) {
	compressedContent, err := compress(msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodingMsg, err)
	}
	return c.split(compressedContent)
}

func (c *chunkingCompressor) split(bytes []byte) ([][]byte, error) {
	splitCompressedContent := splitByteArray(bytes, c.maxChunkSize)

	if len(splitCompressedContent) == 0 {
		return nil, ErrCompressionProducedNoChunks // should never happen
	}
	return splitCompressedContent, nil
}

func compress(msg *types.Message) ([]byte, error) {
	content, err := msg.MarshalContent()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodingMsg, err)
	}
	return compression.CompressBytes(content), nil
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

type noopCompressor struct{}

func (*noopCompressor) Compress(_ *types.Message) ([][]byte, error) {
	return [][]byte{{}}, nil
}

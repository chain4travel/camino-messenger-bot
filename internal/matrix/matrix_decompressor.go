/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"fmt"
	"sort"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
)

var ErrUnmarshalContent = fmt.Errorf("failed to unmarshal content")

func assembleAndDecompressCaminoMatrixMessages(messages []CaminoMatrixMessage) (CaminoMatrixMessage, error) {
	var compressedPayloads [][]byte

	// chunks have to be sorted
	sort.Sort(ByChunkIndex(messages))
	for _, msg := range messages {
		compressedPayloads = append(compressedPayloads, msg.CompressedContent)
	}

	// assemble chunks and decompress content
	originalContent, err := assembleAndDecompress(compressedPayloads)
	if err != nil {
		return CaminoMatrixMessage{}, fmt.Errorf("failed to assemble and decompress camino matrix msg: %v", err)
	}

	msg := CaminoMatrixMessage{
		MessageEventContent: messages[0].MessageEventContent,
		Metadata:            messages[0].Metadata,
	}
	err = msg.UnmarshalContent(originalContent)
	if err != nil {
		return CaminoMatrixMessage{}, fmt.Errorf("%w: %w %v", ErrUnmarshalContent, err, msg.MsgType)
	}

	return msg, nil
}

func assembleAndDecompress(src [][]byte) ([]byte, error) {
	return compression.Decompress(assembleByteArray(src))
}

func assembleByteArray(src [][]byte) []byte {
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

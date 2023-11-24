/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"fmt"
	"sort"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
)

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
	switch messaging.MessageType(msg.MsgType).Category() {
	case messaging.Request,
		messaging.Response:
		msg.UnmarshalContent(originalContent)
	default:
		return CaminoMatrixMessage{}, fmt.Errorf("could not categorize unknown message type: %v", msg.MsgType)
	}

	return msg, nil
}
func assembleAndDecompress(src [][]byte) ([]byte, error) {
	return compression.Decompress(assembleByteArray(src))
}

func assembleByteArray(src [][]byte) []byte {
	var result []byte
	for _, slice := range src {
		result = append(result, slice...)
	}
	return result
}

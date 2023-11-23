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
	"github.com/golang/protobuf/proto"
)

func assembleAndDecompressCaminoMatrixMessages(messagePayloads []caminoMsgEventPayload) (messaging.Message, error) {
	var compressedPayloads [][]byte

	// chunks have to be sorted
	sort.Sort(ByChunkIndex(messagePayloads))
	for _, msg := range messagePayloads {
		compressedPayloads = append(compressedPayloads, msg.caminoMatrixMsg.CompressedContent)
	}

	// assemble chunks and decompress content
	originalContent, err := assembleAndDecompress(compressedPayloads)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("failed to assemble and decompress camino matrix msg: %v", err)
	}

	msg := CaminoMatrixMessage{}
	switch messaging.MessageType(msg.MsgType).Category() {
	case messaging.Request:
		proto.Unmarshal(originalContent, &msg.Content.RequestContent)
	case messaging.Response:
		proto.Unmarshal(originalContent, &msg.Content.ResponseContent)
	default:
		return messaging.Message{}, fmt.Errorf("could not categorize unknown message type: %v", msg.MsgType)
	}

	return messaging.Message{Content: msg.Content, Type: messaging.MessageType(msg.MsgType), Metadata: msg.Metadata}, nil
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

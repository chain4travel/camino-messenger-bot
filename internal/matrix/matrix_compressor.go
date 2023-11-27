/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"maunium.net/go/mautrix/event"
)

func compressAndSplitCaminoMatrixMsg(msg messaging.Message) ([]CaminoMatrixMessage, error) {
	var messages []CaminoMatrixMessage

	var (
		bytes []byte
		err   error
	)
	switch msg.Type.Category() {
	case messaging.Request,
		messaging.Response:
		bytes, err = msg.MarshalContent()
	default:
		return nil, fmt.Errorf("could not categorize unknown message type: %v", msg.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("error while encoding msg for compression: %v", err)
	}

	splitCompressedContent := compressAndSplit(bytes)

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

func compressAndSplit(bytes []byte) [][]byte {
	return splitByteArray(compression.Compress(bytes), compression.MaxChunkSize)
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

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
	"github.com/golang/protobuf/proto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func compressAndSplitCaminoMatrixMsg(roomID id.RoomID, msg messaging.Message) ([]messageEventPayload, error) {
	var messageEvents []messageEventPayload

	var (
		bytes []byte
		err   error
	)
	switch msg.Type.Category() {
	case messaging.Request:
		bytes, err = proto.Marshal(&msg.Content.RequestContent)
	case messaging.Response:
		bytes, err = proto.Marshal(&msg.Content.ResponseContent)
	default:
		return nil, fmt.Errorf("unknown message category: %v", msg.Type.Category())
	}
	if err != nil {
		return nil, fmt.Errorf("error while encoding msg for compression: %v", err)
	}

	splitCompressedContent := compressAndSplit(bytes)

	// add first chunk to messageEvents slice
	{
		caminoMatrixMsg := CaminoMatrixMessage{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            msg.Metadata,
		}
		caminoMatrixMsg.Metadata.NumberOfChunks = uint(len(splitCompressedContent))
		caminoMatrixMsg.CompressedContent = splitCompressedContent[0]
		messageEvents = append(messageEvents, messageEventPayload{
			roomID:      roomID,
			eventType:   C4TMessage,
			contentJSON: caminoMatrixMsg,
		})
	}

	// if multiple chunks were produced upon compression, add them to messageEvents slice
	for i, chunk := range splitCompressedContent[1:] {
		messageEvents = append(messageEvents, messageEventPayload{
			roomID:    roomID,
			eventType: C4TMessage,
			contentJSON: CaminoMatrixMessage{
				MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
				Metadata:            metadata.Metadata{RequestID: msg.Metadata.RequestID, ChunkIndex: uint(i + 1)},
				CompressedContent:   chunk,
			},
		})
	}

	return messageEvents, nil
}

func compressAndSplit(bytes []byte) [][]byte {
	return split(compression.Compress(bytes))
}
func split(bytes []byte) [][]byte {
	//TODO implement
	var chunks [][]byte
	chunks = append(chunks, bytes)
	return chunks
}

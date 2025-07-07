// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/tracing"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var testKey *ecdsa.PrivateKey

func init() {
	var err error
	testKey, err = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(fmt.Errorf("failed to generate test key: %w", err))
	}
}

func TestTryCompleteMessageWithFirstChunk(t *testing.T) {
	logger := zap.NewNop().Sugar()
	botKey := testKey

	messageID := "message-id"
	messageSignature := []byte("signature")

	networkFeeCheque := cheques.SignedCheque{Cheque: cheques.Cheque{
		FromCMAccount: common.Address{1},
	}}

	// we will always expect this message chunk to be present in the map unchanged in addition to case-specific expects
	otherMessageID := "other-message-id"
	otherChunkedMessage := func() *chunkedMessage { // to make copies, not references
		return &chunkedMessage{
			chunksCount:   4,
			signature:     []byte("other-signature"),
			fromCMAccount: common.Address{2},
			chunks: []messageChunk{
				{index: 0, data: []byte("other-chunk0")},
				{index: 1, data: []byte("other-chunk1")},
			},
		}
	}

	tests := map[string]struct {
		msgEventContent         *matrix.SignedMessageEventContent
		existingChunkedMessages map[string]*chunkedMessage
		expectedChunkedMessages map[string]*chunkedMessage
		expectedMessage         messaging.EncodedSignedMessageWithSender
		expectedComplete        bool
	}{
		"Single-chunk message": {
			msgEventContent: &matrix.SignedMessageEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("single chunk data"),
				},
				ChunksCount:      1,
				NetworkFeeCheque: networkFeeCheque,
				Signature:        messageSignature,
			},
			expectedMessage: messaging.EncodedSignedMessageWithSender{
				Message: messaging.EncodedSignedMessage{
					ChunkedEncodedMessage: [][]byte{[]byte("single chunk data")},
					Signature:             messageSignature,
				},
				SenderCMAccountAddress: networkFeeCheque.Cheque.FromCMAccount,
			},
			expectedComplete: true,
		},
		"First chunk is actually first": {
			msgEventContent: &matrix.SignedMessageEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk0"),
				},
				ChunksCount:      3,
				NetworkFeeCheque: networkFeeCheque,
				Signature:        messageSignature,
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunksCount:   3,
					signature:     messageSignature,
					fromCMAccount: networkFeeCheque.Cheque.FromCMAccount,
					chunks: []messageChunk{
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
		},
		"First chunk is second": {
			msgEventContent: &matrix.SignedMessageEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk0"),
				},
				ChunksCount:      3,
				NetworkFeeCheque: networkFeeCheque,
				Signature:        messageSignature,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunksCount:   3,
					signature:     messageSignature,
					fromCMAccount: networkFeeCheque.Cheque.FromCMAccount,
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
		},
		"First chunk is last": {
			msgEventContent: &matrix.SignedMessageEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk0"),
				},
				ChunksCount:      3,
				NetworkFeeCheque: networkFeeCheque,
				Signature:        messageSignature,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 2, data: []byte("chunk2")},
					},
				},
			},
			expectedMessage: messaging.EncodedSignedMessageWithSender{
				Message: messaging.EncodedSignedMessage{
					ChunkedEncodedMessage: [][]byte{[]byte("chunk0"), []byte("chunk1"), []byte("chunk2")},
					Signature:             messageSignature,
				},
				SenderCMAccountAddress: networkFeeCheque.Cheque.FromCMAccount,
			},
			expectedComplete: true,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			matrixClient := NewMockClient(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeSignedMessage, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessageChunk, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(event.StateMember, gomock.Any())

			tracer, err := tracing.NewNoOpTracer()
			require.NoError(t, err)

			matrixMessenger, err := NewMessenger(
				logger,
				tracer,
				matrixClient,
				botKey,
				id.UserID("botUserID"),
			)
			require.NoError(t, err)
			matrixMessengerImpl := matrixMessenger.(*messenger)

			for msgID, chunkedMessage := range tt.existingChunkedMessages {
				matrixMessengerImpl.chunkedMessages[msgID] = chunkedMessage
			}

			if tt.expectedChunkedMessages == nil {
				tt.expectedChunkedMessages = make(map[string]*chunkedMessage)
			}

			// we expect the other message to be present in the map unchanged
			matrixMessengerImpl.chunkedMessages[otherMessageID] = otherChunkedMessage()
			tt.expectedChunkedMessages[otherMessageID] = otherChunkedMessage()

			message, completed := matrixMessengerImpl.tryCompleteMessageWithFirstChunk(tt.msgEventContent)
			require.Equal(t, tt.expectedComplete, completed)
			require.Equal(t, tt.expectedMessage, message)
			require.Equal(t, tt.expectedChunkedMessages, matrixMessengerImpl.chunkedMessages)
		})
	}
}

func TestTryCompleteMessage(t *testing.T) {
	logger := zap.NewNop().Sugar()
	botKey := testKey

	messageID := "message-id"
	messageSignature := []byte("signature")
	senderCMAccount := common.Address{1}

	// we will always expect this message chunk to be present in the map unchanged in addition to case-specific expects
	otherMessageID := "other-message-id"
	otherChunkedMessage := func() *chunkedMessage { // to make copies, not references
		return &chunkedMessage{
			chunksCount:   4,
			signature:     []byte("other-signature"),
			fromCMAccount: common.Address{2},
			chunks: []messageChunk{
				{index: 0, data: []byte("other-chunk0")},
				{index: 1, data: []byte("other-chunk1")},
			},
		}
	}

	tests := map[string]struct {
		msgEventContent         *matrix.MessageChunkEventContent
		existingChunkedMessages map[string]*chunkedMessage
		expectedChunkedMessages map[string]*chunkedMessage
		expectedMessage         messaging.EncodedSignedMessageWithSender
		expectedComplete        bool
	}{
		"Chunk is first": {
			msgEventContent: &matrix.MessageChunkEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk1"),
				},
				ChunkIndex: 1,
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
		},
		"Chunk is second": {
			msgEventContent: &matrix.MessageChunkEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk2"),
				},
				ChunkIndex: 2,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 2, data: []byte("chunk2")},
					},
				},
			},
		},
		"Chunk is last": {
			msgEventContent: &matrix.MessageChunkEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      []byte("chunk1"),
				},
				ChunkIndex: 1,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				messageID: {
					chunksCount:   3,
					signature:     messageSignature,
					fromCMAccount: senderCMAccount,
					chunks: []messageChunk{
						{index: 2, data: []byte("chunk2")},
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
			expectedMessage: messaging.EncodedSignedMessageWithSender{
				Message: messaging.EncodedSignedMessage{
					ChunkedEncodedMessage: [][]byte{[]byte("chunk0"), []byte("chunk1"), []byte("chunk2")},
					Signature:             messageSignature,
				},
				SenderCMAccountAddress: senderCMAccount,
			},
			expectedComplete: true,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			matrixClient := NewMockClient(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeSignedMessage, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessageChunk, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(event.StateMember, gomock.Any())

			tracer, err := tracing.NewNoOpTracer()
			require.NoError(t, err)

			matrixMessenger, err := NewMessenger(
				logger,
				tracer,
				matrixClient,
				botKey,
				id.UserID("botUserID"),
			)
			require.NoError(t, err)
			matrixMessengerImpl := matrixMessenger.(*messenger)

			for msgID, chunkedMessage := range tt.existingChunkedMessages {
				matrixMessengerImpl.chunkedMessages[msgID] = chunkedMessage
			}

			if tt.expectedChunkedMessages == nil {
				tt.expectedChunkedMessages = make(map[string]*chunkedMessage)
			}

			// we expect the other message to be present in the map unchanged
			matrixMessengerImpl.chunkedMessages[otherMessageID] = otherChunkedMessage()
			tt.expectedChunkedMessages[otherMessageID] = otherChunkedMessage()

			message, completed := matrixMessengerImpl.tryCompleteMessage(tt.msgEventContent)
			require.Equal(t, tt.expectedComplete, completed)
			require.Equal(t, tt.expectedMessage, message)
			require.Equal(t, tt.expectedChunkedMessages, matrixMessengerImpl.chunkedMessages)
		})
	}
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
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

	requestID := "message-id"
	serviceFeeCheque := &cheques.SignedCheque{Signature: []byte("service-fee-signature")}
	networkFeeCheque := &cheques.SignedCheque{Signature: []byte("network-fee-signature")}
	timestamps := metadata.Timestamps{"1": 9876543210}

	// we will always expect this message chunk to be present in the map unchanged in addition to case-specific expects
	otherRequestID := "other-message-id"
	otherChunkedMessage := func() *chunkedMessage { // to make copies, not references
		return &chunkedMessage{
			timestamps:       metadata.Timestamps{"1-other": 1234567890},
			serviceFeeCheque: &cheques.SignedCheque{Signature: []byte("other-service-fee-signature")},
			networkFeeCheque: &cheques.SignedCheque{Signature: []byte("other-network-fee-signature")},
			chunksCount:      4,
			msgType:          generated.AccommodationProductInfoServiceV1Request,
			chunks: []messageChunk{
				{index: 0, data: []byte("other-chunk0")},
				{index: 1, data: []byte("other-chunk1")},
			},
		}
	}

	content := pingv1.PingRequest{
		PingMessage: "ping message",
	}
	contentBytes, err := proto.Marshal(&content)
	require.NoError(t, err)

	tests := map[string]struct {
		decompressor            func(*gomock.Controller) *compression.MockDecompressor
		msgEventContent         *matrix.MessageEventContent
		existingChunkedMessages map[string]*chunkedMessage
		expectedChunkedMessages map[string]*chunkedMessage
		expectedMessage         types.Message
		expectedComplete        bool
		expectedError           error
	}{
		"Single-chunk message": {
			decompressor: func(ctrl *gomock.Controller) *compression.MockDecompressor {
				decompressor := compression.NewMockDecompressor(ctrl)
				decompressor.EXPECT().Decompress([]byte("single chunk data")).Return(contentBytes, nil)
				return decompressor
			},
			msgEventContent: &matrix.MessageEventContent{
				MessageChunkEventContent: matrix.MessageChunkEventContent{
					RequestID: requestID,
					Data:      []byte("single chunk data"),
				},
				MsgType:          generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: *networkFeeCheque,
				ChunksCount:      1,
			},
			expectedMessage: types.Message{
				RequestID:        requestID,
				Type:             generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: networkFeeCheque,
				Content:          proto.Clone(&content),
			},
			expectedComplete: true,
		},
		"First chunk is actually first": {
			msgEventContent: &matrix.MessageEventContent{
				MessageChunkEventContent: matrix.MessageChunkEventContent{
					RequestID: requestID,
					Data:      []byte("chunk0"),
				},
				MsgType:          generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: *networkFeeCheque,
				ChunksCount:      3,
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					msgType:          generated.PingServiceV1Request,
					chunksCount:      3,
					timestamps:       timestamps,
					serviceFeeCheque: serviceFeeCheque,
					networkFeeCheque: networkFeeCheque,
					chunks: []messageChunk{
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
		},
		"First chunk is second": {
			msgEventContent: &matrix.MessageEventContent{
				MessageChunkEventContent: matrix.MessageChunkEventContent{
					RequestID: requestID,
					Data:      []byte("chunk0"),
				},
				MsgType:          generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: *networkFeeCheque,
				ChunksCount:      3,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					msgType:          generated.PingServiceV1Request,
					chunksCount:      3,
					timestamps:       timestamps,
					serviceFeeCheque: serviceFeeCheque,
					networkFeeCheque: networkFeeCheque,
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
		},
		"First chunk is last": {
			decompressor: func(ctrl *gomock.Controller) *compression.MockDecompressor {
				decompressor := compression.NewMockDecompressor(ctrl)
				decompressor.EXPECT().Decompress([]byte("chunk0chunk1chunk2")).Return(contentBytes, nil)
				return decompressor
			},
			msgEventContent: &matrix.MessageEventContent{
				MessageChunkEventContent: matrix.MessageChunkEventContent{
					RequestID: requestID,
					Data:      []byte("chunk0"),
				},
				MsgType:          generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: *networkFeeCheque,
				ChunksCount:      3,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 2, data: []byte("chunk2")},
					},
				},
			},
			expectedMessage: types.Message{
				RequestID:        requestID,
				Type:             generated.PingServiceV1Request,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: networkFeeCheque,
				Content:          proto.Clone(&content),
			},
			expectedComplete: true,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			matrixClient := NewMockClient(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessage, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessageChunk, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(event.StateMember, gomock.Any())

			if tt.decompressor == nil {
				tt.decompressor = compression.NewMockDecompressor
			}

			matrixMessenger, err := NewMessenger(
				logger,
				matrixClient,
				tt.decompressor(ctrl),
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
			matrixMessengerImpl.chunkedMessages[otherRequestID] = otherChunkedMessage()
			tt.expectedChunkedMessages[otherRequestID] = otherChunkedMessage()

			message, completed, err := matrixMessengerImpl.tryCompleteMessageWithFirstChunk(tt.msgEventContent)
			require.ErrorIs(t, err, tt.expectedError)
			require.Equal(t, tt.expectedComplete, completed)
			if completed {
				require.True(t, proto.Equal(tt.expectedMessage.Content, message.Content))
				proto.Reset(tt.expectedMessage.Content)
				proto.Reset(message.Content)
			}
			require.Equal(t, tt.expectedMessage, message)
			require.Equal(t, tt.expectedChunkedMessages, matrixMessengerImpl.chunkedMessages)
		})
	}
}

func TestTryCompleteMessage(t *testing.T) {
	logger := zap.NewNop().Sugar()
	botKey := testKey

	requestID := "message-id"
	serviceFeeCheque := &cheques.SignedCheque{Signature: []byte("service-fee-signature")}
	networkFeeCheque := &cheques.SignedCheque{Signature: []byte("network-fee-signature")}
	timestamps := metadata.Timestamps{"1": 9876543210}

	// we will always expect this message chunk to be present in the map unchanged in addition to case-specific expects
	otherRequestID := "other-message-id"
	otherChunkedMessage := func() *chunkedMessage { // to make copies, not references
		return &chunkedMessage{
			timestamps:       metadata.Timestamps{"1-other": 1234567890},
			serviceFeeCheque: &cheques.SignedCheque{Signature: []byte("other-service-fee-signature")},
			networkFeeCheque: &cheques.SignedCheque{Signature: []byte("other-network-fee-signature")},
			chunksCount:      4,
			msgType:          generated.AccommodationProductInfoServiceV1Request,
			chunks: []messageChunk{
				{index: 0, data: []byte("other-chunk0")},
				{index: 1, data: []byte("other-chunk1")},
			},
		}
	}

	content := pingv1.PingRequest{
		PingMessage: "ping message",
	}
	contentBytes, err := proto.Marshal(&content)
	require.NoError(t, err)

	tests := map[string]struct {
		decompressor            func(*gomock.Controller) *compression.MockDecompressor
		msgEventContent         *matrix.MessageChunkEventContent
		existingChunkedMessages map[string]*chunkedMessage
		expectedChunkedMessages map[string]*chunkedMessage
		expectedMessage         types.Message
		expectedComplete        bool
		expectedError           error
	}{
		"Chunk is first": {
			msgEventContent: &matrix.MessageChunkEventContent{
				RequestID:  requestID,
				Data:       []byte("chunk1"),
				ChunkIndex: 1,
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
		},
		"Chunk is second": {
			msgEventContent: &matrix.MessageChunkEventContent{
				RequestID:  requestID,
				Data:       []byte("chunk2"),
				ChunkIndex: 2,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
					},
				},
			},
			expectedChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunks: []messageChunk{
						{index: 1, data: []byte("chunk1")},
						{index: 2, data: []byte("chunk2")},
					},
				},
			},
		},
		"Chunk is last": {
			decompressor: func(ctrl *gomock.Controller) *compression.MockDecompressor {
				decompressor := compression.NewMockDecompressor(ctrl)
				decompressor.EXPECT().Decompress([]byte("chunk0chunk1chunk2")).Return(contentBytes, nil)
				return decompressor
			},
			msgEventContent: &matrix.MessageChunkEventContent{
				RequestID:  requestID,
				Data:       []byte("chunk1"),
				ChunkIndex: 1,
			},
			existingChunkedMessages: map[string]*chunkedMessage{
				requestID: {
					chunksCount:      3,
					msgType:          generated.PingServiceV1Request,
					timestamps:       timestamps,
					serviceFeeCheque: serviceFeeCheque,
					networkFeeCheque: networkFeeCheque,
					chunks: []messageChunk{
						{index: 2, data: []byte("chunk2")},
						{index: 0, data: []byte("chunk0")},
					},
				},
			},
			expectedMessage: types.Message{
				RequestID:        requestID,
				Timestamps:       timestamps,
				ServiceFeeCheque: serviceFeeCheque,
				NetworkFeeCheque: networkFeeCheque,
				Type:             generated.PingServiceV1Request,
				Content:          proto.Clone(&content),
			},
			expectedComplete: true,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			matrixClient := NewMockClient(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessage, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessageChunk, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(event.StateMember, gomock.Any())

			if tt.decompressor == nil {
				tt.decompressor = compression.NewMockDecompressor
			}

			matrixMessenger, err := NewMessenger(
				logger,
				matrixClient,
				tt.decompressor(ctrl),
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
			matrixMessengerImpl.chunkedMessages[otherRequestID] = otherChunkedMessage()
			tt.expectedChunkedMessages[otherRequestID] = otherChunkedMessage()

			message, completed, err := matrixMessengerImpl.tryCompleteMessage(tt.msgEventContent)
			require.ErrorIs(t, err, tt.expectedError)
			require.Equal(t, tt.expectedComplete, completed)
			if completed {
				require.True(t, proto.Equal(tt.expectedMessage.Content, message.Content))
				proto.Reset(tt.expectedMessage.Content)
				proto.Reset(message.Content)
			}
			require.Equal(t, tt.expectedMessage, message)
			require.Equal(t, tt.expectedChunkedMessages, matrixMessengerImpl.chunkedMessages)
		})
	}
}

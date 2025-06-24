// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
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

func TestTryAssembleMessage(t *testing.T) {
	logger := zap.NewNop().Sugar()
	botKey := testKey
	testError := errors.New("test error")

	msg := types.Message{
		Metadata: metadata.Metadata{
			RequestID:          "testRequestID",
			RecipientCMAccount: "0xADDRESS",
		},
		Type:    generated.PingServiceV1Response,
		Content: &pingv1.PingResponse{PingMessage: "pong"},
	}
	contentBytes, err := msg.MarshalContent()
	require.NoError(t, err)

	compressedContentBytes := []byte{'c', 'o', 'm', 'p', 'r', 'e', 's', 's', 'e', 'd'}

	tests := map[string]struct {
		decompressor                     func(c *gomock.Controller) *compression.MockDecompressor
		existingMsgEventContents         map[string][]*matrix.CaminoMatrixMessageEventContent
		msgEventContent                  *matrix.CaminoMatrixMessageEventContent
		expectedExistingMsgEventContents map[string][]*matrix.CaminoMatrixMessageEventContent
		expectedMessage                  types.Message
		expectedComplete                 bool
		expectedErr                      error
	}{
		"Decoder failed to decompress": {
			decompressor: func(c *gomock.Controller) *compression.MockDecompressor {
				d := compression.NewMockDecompressor(c)
				d.EXPECT().Decompress(compressedContentBytes).Return(nil, testError)
				return d
			},
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 1,
				},
				CompressedContent: compressedContentBytes,
			},
			expectedErr: errDecompressFailed,
		},
		"Unknown message type": {
			decompressor: func(c *gomock.Controller) *compression.MockDecompressor {
				d := compression.NewMockDecompressor(c)
				d.EXPECT().Decompress(compressedContentBytes).Return([]byte{}, nil)
				return d
			},
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 1,
				},
				CompressedContent: compressedContentBytes,
			},
			expectedErr: errUnmarshalContent,
		},
		"OK: Empty input": {
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{},
			expectedExistingMsgEventContents: map[string][]*matrix.CaminoMatrixMessageEventContent{
				"": {{}},
			},
		},
		"OK: Single chunk message": {
			decompressor: func(c *gomock.Controller) *compression.MockDecompressor {
				d := compression.NewMockDecompressor(c)
				d.EXPECT().Decompress(compressedContentBytes).Return(contentBytes, nil)
				return d
			},
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{
				MessageEventContent: event.MessageEventContent{
					MsgType: event.MessageType(generated.PingServiceV1Response),
				},
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 1,
				},
				CompressedContent: compressedContentBytes,
			},
			expectedMessage: types.Message{
				Type:    generated.PingServiceV1Response,
				Content: proto.Clone(msg.Content),
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 1,
				},
			},
			expectedComplete: true,
		},
		"OK: 3-chunk message, first chunk": {
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 3,
					ChunkIndex:     1,
				},
				CompressedContent: compressedContentBytes[3:5],
			},
			expectedExistingMsgEventContents: map[string][]*matrix.CaminoMatrixMessageEventContent{
				msg.Metadata.RequestID: {
					{
						Metadata: metadata.Metadata{
							RequestID:      msg.Metadata.RequestID,
							NumberOfChunks: 3,
							ChunkIndex:     1,
						},
						CompressedContent: compressedContentBytes[3:5],
					},
				},
			},
		},
		"OK: 3-chunk message, not first, but not last chunk": {
			existingMsgEventContents: map[string][]*matrix.CaminoMatrixMessageEventContent{
				msg.Metadata.RequestID: {{
					Metadata: metadata.Metadata{
						RequestID:      msg.Metadata.RequestID,
						NumberOfChunks: 3,
						ChunkIndex:     1,
					},
					CompressedContent: compressedContentBytes[3:5],
				}},
			},
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{
				Metadata: metadata.Metadata{
					RequestID:          msg.Metadata.RequestID,
					NumberOfChunks:     3,
					ChunkIndex:         0,
					RecipientCMAccount: msg.Metadata.RecipientCMAccount,
				},
				CompressedContent: compressedContentBytes[0:3],
			},
			expectedExistingMsgEventContents: map[string][]*matrix.CaminoMatrixMessageEventContent{
				msg.Metadata.RequestID: {
					{
						Metadata: metadata.Metadata{
							RequestID:      msg.Metadata.RequestID,
							NumberOfChunks: 3,
							ChunkIndex:     1,
						},
						CompressedContent: compressedContentBytes[3:5],
					},
					{
						Metadata: metadata.Metadata{
							RequestID:          msg.Metadata.RequestID,
							NumberOfChunks:     3,
							ChunkIndex:         0,
							RecipientCMAccount: msg.Metadata.RecipientCMAccount,
						},
						CompressedContent: compressedContentBytes[0:3],
					},
				},
			},
		},
		"OK: 3-chunk message, last chunk": {
			decompressor: func(c *gomock.Controller) *compression.MockDecompressor {
				d := compression.NewMockDecompressor(c)
				d.EXPECT().Decompress(compressedContentBytes).Return(contentBytes, nil)
				return d
			},
			existingMsgEventContents: map[string][]*matrix.CaminoMatrixMessageEventContent{
				msg.Metadata.RequestID: {
					{
						Metadata: metadata.Metadata{
							RequestID:      msg.Metadata.RequestID,
							NumberOfChunks: 3,
							ChunkIndex:     1,
						},
						CompressedContent: compressedContentBytes[3:5],
					},
					{
						MessageEventContent: event.MessageEventContent{
							MsgType: event.MessageType(generated.PingServiceV1Response),
						},
						Metadata: metadata.Metadata{
							RequestID:          msg.Metadata.RequestID,
							NumberOfChunks:     3,
							ChunkIndex:         0,
							RecipientCMAccount: msg.Metadata.RecipientCMAccount,
						},
						CompressedContent: compressedContentBytes[0:3],
					},
				},
			},
			msgEventContent: &matrix.CaminoMatrixMessageEventContent{ // last message
				Metadata: metadata.Metadata{
					RequestID:      msg.Metadata.RequestID,
					NumberOfChunks: 3,
					ChunkIndex:     2,
				},
				CompressedContent: compressedContentBytes[5:],
			},
			expectedMessage: types.Message{
				Type:    generated.PingServiceV1Response,
				Content: proto.Clone(msg.Content),
				Metadata: metadata.Metadata{
					RequestID:          msg.Metadata.RequestID,
					NumberOfChunks:     3,
					RecipientCMAccount: msg.Metadata.RecipientCMAccount,
				},
			},
			expectedComplete: true,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			matrixClient := NewMockClient(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeC4TMessage, gomock.Any())
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

			if tt.existingMsgEventContents != nil {
				matrixMessengerImpl.messages = tt.existingMsgEventContents
			}

			message, completed, err := matrixMessengerImpl.tryAssembleMessage(tt.msgEventContent)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, tt.expectedComplete, completed)

			if tt.expectedComplete {
				require.True(t, proto.Equal(tt.expectedMessage.Content, message.Content))
				proto.Reset(tt.expectedMessage.Content)
				proto.Reset(message.Content)
				require.Equal(t, tt.expectedMessage, message)
			} else {
				require.Empty(t, message)
			}

			if tt.expectedExistingMsgEventContents == nil {
				tt.expectedExistingMsgEventContents = make(map[string][]*matrix.CaminoMatrixMessageEventContent)
			}
			require.Equal(t, tt.expectedExistingMsgEventContents, matrixMessengerImpl.messages)
		})
	}
}

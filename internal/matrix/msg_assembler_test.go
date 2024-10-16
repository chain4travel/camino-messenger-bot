/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/matrix"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"maunium.net/go/mautrix/event"
)

func TestAssembleMessage(t *testing.T) {
	pingResponseMsg := types.Message{
		Type:    generated.PingServiceV1Response,
		Content: &pingv1.PingResponse{PingMessage: "pong"},
	}
	type fields struct {
		partialMessages map[string][]*matrix.CaminoMatrixMessage
	}

	type args struct {
		msg *matrix.CaminoMatrixMessage
	}

	// mocks
	ctrl := gomock.NewController(t)
	mockedDecompressor := compression.NewMockDecompressor(ctrl)

	tests := map[string]struct {
		fields     fields
		args       args
		prepare    func()
		want       *matrix.CaminoMatrixMessage
		isComplete bool
		err        error
	}{
		"err: decoder failed to decompress": {
			args: args{
				msg: &matrix.CaminoMatrixMessage{
					Metadata: metadata.Metadata{
						RequestID:      "test",
						NumberOfChunks: 1,
					},
				},
			},
			prepare: func() {
				mockedDecompressor.EXPECT().Decompress(gomock.Any()).Times(1).Return(nil, ErrDecompressFailed)
			},
			isComplete: false,
			err:        ErrDecompressFailed,
		},
		"err: unknown message type": {
			args: args{
				msg: &matrix.CaminoMatrixMessage{
					Metadata: metadata.Metadata{
						RequestID:      "test",
						NumberOfChunks: 1,
					},
				},
			},
			prepare: func() {
				mockedDecompressor.EXPECT().Decompress(gomock.Any()).Times(1).Return([]byte{}, nil)
			},
			isComplete: false,
			err:        ErrUnmarshalContent,
		},
		"empty input": {
			fields: fields{
				partialMessages: map[string][]*matrix.CaminoMatrixMessage{},
			},
			args: args{
				msg: &matrix.CaminoMatrixMessage{},
			},
			isComplete: false,
			err:        nil,
		},
		"partial message delivery [metadata number fo chunks do not match provided messages]": {
			fields: fields{
				partialMessages: map[string][]*matrix.CaminoMatrixMessage{},
			},
			args: args{
				msg: &matrix.CaminoMatrixMessage{
					Metadata: metadata.Metadata{
						RequestID:      "test",
						NumberOfChunks: 2,
					},
				},
			},
			isComplete: false,
			err:        nil,
		},
		"success: single chunk message": {
			fields: fields{
				partialMessages: map[string][]*matrix.CaminoMatrixMessage{},
			},
			args: args{
				msg: &matrix.CaminoMatrixMessage{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(generated.PingServiceV1Response),
					},
					Metadata: metadata.Metadata{
						RequestID:      "id",
						NumberOfChunks: 1,
					},
				}, // last message
			},
			prepare: func() {
				msg := pingResponseMsg
				msgBytes, err := msg.MarshalContent()
				require.NoError(t, err)
				mockedDecompressor.EXPECT().Decompress(gomock.Any()).Times(1).Return(msgBytes, nil)
			},
			want: &matrix.CaminoMatrixMessage{
				Metadata: metadata.Metadata{
					RequestID:      "id",
					NumberOfChunks: 1,
				},
				MessageEventContent: event.MessageEventContent{
					MsgType: event.MessageType(generated.PingServiceV1Response),
				},
				Content: pingResponseMsg.Content,
			},
			isComplete: true,
			err:        nil,
		},
		"success: multi-chunk message": {
			fields: fields{
				partialMessages: map[string][]*matrix.CaminoMatrixMessage{"id": {
					// only 2 chunks because the last one is passed as the last argument triggering the call of AssembleMessage
					// msgType is necessary only for 1st chunk
					{MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(generated.PingServiceV1Response)}}, {},
				}},
			},
			args: args{
				msg: &matrix.CaminoMatrixMessage{
					Metadata: metadata.Metadata{
						RequestID:      "id",
						NumberOfChunks: 3,
					},
				}, // last message
			},
			prepare: func() {
				msg := pingResponseMsg
				msgBytes, err := msg.MarshalContent()
				require.NoError(t, err)
				mockedDecompressor.EXPECT().Decompress(gomock.Any()).Times(1).Return(msgBytes, nil)
			},
			want: &matrix.CaminoMatrixMessage{
				MessageEventContent: event.MessageEventContent{
					MsgType: event.MessageType(generated.PingServiceV1Response),
				},
				Content: pingResponseMsg.Content,
			},
			isComplete: true,
			err:        nil,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			a := &messageAssembler{
				partialMessages: tt.fields.partialMessages,
				decompressor:    mockedDecompressor,
			}
			if tt.prepare != nil {
				tt.prepare()
			}
			got, isComplete, err := a.AssembleMessage(tt.args.msg)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.isComplete, isComplete, "AssembleMessage() isComplete = %v, expRoomID %v", isComplete, tt.isComplete)

			// Reset the response content to avoid comparisons of pb fields like sizeCache
			if tt.want != nil && got != nil {
				proto.Reset(tt.want.Content)
				proto.Reset(got.Content)
			}
			require.Equal(t, tt.want, got, "AssembleMessage() got = %v, expRoomID %v", got, tt.want)
		})
	}
}

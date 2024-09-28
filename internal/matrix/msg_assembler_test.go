/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"testing"

	"maunium.net/go/mautrix/event"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/matrix"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAssembleMessage(t *testing.T) {
	plainActivitySearchResponseMsg := messaging.Message{
		Type: messaging.ActivitySearchResponse,
		Content: messaging.MessageContent{
			ResponseContent: messaging.ResponseContent{
				ActivitySearchResponse: &activityv2.ActivitySearchResponse{
					Results: []*activityv2.ActivitySearchResult{
						{Info: &activityv2.Activity{ServiceCode: "test"}},
					},
				},
			},
		},
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
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						RequestID:      "id",
						NumberOfChunks: 1,
					},
				}, // last message
			},
			prepare: func() {
				msg := plainActivitySearchResponseMsg
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
					MsgType: event.MessageType(messaging.ActivitySearchResponse),
				},
				Content: plainActivitySearchResponseMsg.Content,
			},
			isComplete: true,
			err:        nil,
		},
		"success: multi-chunk message": {
			fields: fields{
				partialMessages: map[string][]*matrix.CaminoMatrixMessage{"id": {
					// only 2 chunks because the last one is passed as the last argument triggering the call of AssembleMessage
					// msgType is necessary only for 1st chunk
					{MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(messaging.ActivitySearchResponse)}}, {},
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
				msg := plainActivitySearchResponseMsg
				msgBytes, err := msg.MarshalContent()
				require.NoError(t, err)
				mockedDecompressor.EXPECT().Decompress(gomock.Any()).Times(1).Return(msgBytes, nil)
			},
			want: &matrix.CaminoMatrixMessage{
				MessageEventContent: event.MessageEventContent{
					MsgType: event.MessageType(messaging.ActivitySearchResponse),
				},
				Content: plainActivitySearchResponseMsg.Content,
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
				tt.want.Content.ResponseContent.ActivitySearchResponse.Reset()
				got.Content.ResponseContent.ActivitySearchResponse.Reset()
			}
			require.Equal(t, tt.want, got, "AssembleMessage() got = %v, expRoomID %v", got, tt.want)
		})
	}
}

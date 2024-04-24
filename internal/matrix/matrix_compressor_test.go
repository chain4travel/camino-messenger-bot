/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"testing"

	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/stretchr/testify/require"
	"maunium.net/go/mautrix/event"
)

func TestMatrixChunkingCompressorCompress(t *testing.T) {
	type args struct {
		msg     *messaging.Message
		maxSize int
	}
	tests := map[string]struct {
		args args
		want []*CaminoMatrixMessage
		err  error
	}{
		"err: unknown message type": {
			args: args{msg: &messaging.Message{Type: "Unknown"}, maxSize: 5},
			err:  messaging.ErrUnknownMessageType,
		},
		"err: empty message": {
			args: args{msg: &messaging.Message{Type: messaging.ActivitySearchResponse}, maxSize: 5},
			err:  ErrCompressionProducedNoChunks,
		},
		"success: small message compressed without chunking (input<maxSize)": {
			args: args{
				msg: &messaging.Message{
					Type: messaging.ActivitySearchResponse,
					Content: messaging.MessageContent{
						ResponseContent: messaging.ResponseContent{
							ActivitySearchResponse: &activityv1alpha.ActivitySearchResponse{
								Results: []*activityv1alpha.ActivitySearchResult{
									{Info: &activityv1alpha.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 100,
			},
			want: []*CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
				},
			},
		},
		"success: small message compressed without chunking (input=maxSize)": {
			args: args{
				msg: &messaging.Message{
					Type: messaging.ActivitySearchResponse,
					Content: messaging.MessageContent{
						ResponseContent: messaging.ResponseContent{
							ActivitySearchResponse: &activityv1alpha.ActivitySearchResponse{
								Results: []*activityv1alpha.ActivitySearchResult{
									{Info: &activityv1alpha.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 23, // compressed size of msgType=ActivitySearchResponse and serviceCode="test"
			},
			want: []*CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
				},
			},
		},
		"success: large message compressed with chunking (input>maxSize)": {
			args: args{
				msg: &messaging.Message{
					Type: messaging.ActivitySearchResponse,
					Content: messaging.MessageContent{
						ResponseContent: messaging.ResponseContent{
							ActivitySearchResponse: &activityv1alpha.ActivitySearchResponse{
								Results: []*activityv1alpha.ActivitySearchResult{
									{Info: &activityv1alpha.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 22, // < 23 = compressed size of msgType=ActivitySearchResponse and serviceCode="test"
			},
			want: []*CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     0,
					},
				},
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     1,
					},
				},
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			c := &ChunkingCompressor{tt.args.maxSize}
			got, err := c.Compress(tt.args.msg)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, len(got), len(tt.want))
			for i, msg := range got {
				require.Equal(t, msg.MessageEventContent.MsgType, tt.want[i].MsgType)
				require.Equal(t, msg.Metadata.NumberOfChunks, tt.want[i].Metadata.NumberOfChunks)
				require.Equal(t, msg.Metadata.ChunkIndex, tt.want[i].Metadata.ChunkIndex)
				require.NotNil(t, msg.CompressedContent)
			}
		})
	}
}

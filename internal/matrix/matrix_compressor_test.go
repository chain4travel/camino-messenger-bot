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

func TestChunkingCompressorCompress(t *testing.T) {
	type args struct {
		msg     messaging.Message
		maxSize int
	}
	tests := map[string]struct {
		args args
		want []CaminoMatrixMessage
		err  error
	}{
		"err: unknown message type": {
			args: args{msg: messaging.Message{Type: "Unknown"}, maxSize: 5},
			err:  messaging.ErrUnknownMessageType,
		},
		"err: empty message": {
			args: args{msg: messaging.Message{Type: messaging.ActivitySearchResponse}, maxSize: 5},
			err:  ErrCompressionProducedNoChunks,
		},
		"success: small message compressed without chunking (input<maxSize)": {
			args: args{
				msg: messaging.Message{
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
			want: []CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
					CompressedContent: []byte{40, 181, 47, 253, 4, 0, 81, 0, 0, 26, 8, 18, 6, 42, 4, 116, 101, 115, 116, 39, 101, 69, 66},
				},
			},
		},
		"success: small message compressed without chunking (input=maxSize)": {
			args: args{
				msg: messaging.Message{
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
			want: []CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
					CompressedContent: []byte{40, 181, 47, 253, 4, 0, 81, 0, 0, 26, 8, 18, 6, 42, 4, 116, 101, 115, 116, 39, 101, 69, 66},
				},
			},
		},
		"success: large message compressed with chunking (input>maxSize)": {
			args: args{
				msg: messaging.Message{
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
			want: []CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     0,
					},
					CompressedContent: []byte{40, 181, 47, 253, 4, 0, 81, 0, 0, 26, 8, 18, 6, 42, 4, 116, 101, 115, 116, 39, 101, 69},
				},
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     1,
					},
					CompressedContent: []byte{66},
				},
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			c := &ChunkingCompressor{tt.args.maxSize}
			got, err := c.Compress(tt.args.msg)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

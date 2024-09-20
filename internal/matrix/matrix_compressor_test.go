/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"testing"

	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/matrix"
	"github.com/stretchr/testify/require"
	"maunium.net/go/mautrix/event"
)

func compressOnly(msg messaging.Message) []byte {
	compressed, _ := compress(msg) // Ignoring the second return value
	return compressed
}

func TestChunkingCompressorCompress(t *testing.T) {
	type args struct {
		msg     messaging.Message
		maxSize int
	}
	tests := map[string]struct {
		args args
		want []matrix.CaminoMatrixMessage
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
							ActivitySearchResponse: &activityv1.ActivitySearchResponse{
								Results: []*activityv1.ActivitySearchResult{
									{Info: &activityv1.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 100,
			},
			want: []matrix.CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
					CompressedContent: compressOnly(
						messaging.Message{
							Type: messaging.ActivitySearchResponse,
							Content: messaging.MessageContent{
								ResponseContent: messaging.ResponseContent{
									ActivitySearchResponse: &activityv1.ActivitySearchResponse{
										Results: []*activityv1.ActivitySearchResult{
											{Info: &activityv1.Activity{ServiceCode: "test"}},
										},
									},
								},
							},
						}),
				},
			},
		},
		"success: small message compressed without chunking (input=maxSize)": {
			args: args{
				msg: messaging.Message{
					Type: messaging.ActivitySearchResponse,
					Content: messaging.MessageContent{
						ResponseContent: messaging.ResponseContent{
							ActivitySearchResponse: &activityv1.ActivitySearchResponse{
								Results: []*activityv1.ActivitySearchResult{
									{Info: &activityv1.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 23, // compressed size of msgType=ActivitySearchResponse and serviceCode="test"
			},
			want: []matrix.CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 1,
						ChunkIndex:     0,
					},
					CompressedContent: compressOnly(
						messaging.Message{
							Type: messaging.ActivitySearchResponse,
							Content: messaging.MessageContent{
								ResponseContent: messaging.ResponseContent{
									ActivitySearchResponse: &activityv1.ActivitySearchResponse{
										Results: []*activityv1.ActivitySearchResult{
											{Info: &activityv1.Activity{ServiceCode: "test"}},
										},
									},
								},
							},
						}),
				},
			},
		},
		"success: large message compressed with chunking (input>maxSize)": {
			args: args{
				msg: messaging.Message{
					Type: messaging.ActivitySearchResponse,
					Content: messaging.MessageContent{
						ResponseContent: messaging.ResponseContent{
							ActivitySearchResponse: &activityv1.ActivitySearchResponse{
								Results: []*activityv1.ActivitySearchResult{
									{Info: &activityv1.Activity{ServiceCode: "test"}},
								},
							},
						},
					},
				},
				maxSize: 22, // < 23 = compressed size of msgType=ActivitySearchResponse and serviceCode="test"
			},
			want: []matrix.CaminoMatrixMessage{
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     0,
					},
					CompressedContent: func() []byte {
						compressedContent := compressOnly(messaging.Message{
							Type: messaging.ActivitySearchResponse,
							Content: messaging.MessageContent{
								ResponseContent: messaging.ResponseContent{
									ActivitySearchResponse: &activityv1.ActivitySearchResponse{
										Results: []*activityv1.ActivitySearchResult{
											{Info: &activityv1.Activity{ServiceCode: "test"}},
										},
									},
								},
							},
						})
						return compressedContent[:22] // First Chunk
					}(),
				},
				{
					MessageEventContent: event.MessageEventContent{
						MsgType: event.MessageType(messaging.ActivitySearchResponse),
					},
					Metadata: metadata.Metadata{
						NumberOfChunks: 2,
						ChunkIndex:     1,
					},
					CompressedContent: func() []byte {
						compressedContent := compressOnly(messaging.Message{
							Type: messaging.ActivitySearchResponse,
							Content: messaging.MessageContent{
								ResponseContent: messaging.ResponseContent{
									ActivitySearchResponse: &activityv1.ActivitySearchResponse{
										Results: []*activityv1.ActivitySearchResult{
											{Info: &activityv1.Activity{ServiceCode: "test"}},
										},
									},
								},
							},
						})
						return compressedContent[22:] // Second Chunk
					}(),
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

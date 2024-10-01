/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client/generated"
	"github.com/stretchr/testify/require"
)

func compressOnly(t *testing.T, msg *types.Message) []byte {
	compressed, err := compress(msg)
	require.NoError(t, err)
	return compressed
}

func TestChunkingCompressorCompress(t *testing.T) {
	type args struct {
		msg     *types.Message
		maxSize int
	}
	tests := map[string]struct {
		args args
		want [][]byte
		err  error
	}{
		"err: empty message": {
			args: args{msg: &types.Message{Type: generated.PingServiceV1Response}, maxSize: 5},
			err:  ErrCompressionProducedNoChunks,
		},
		"success: small message compressed without chunking (input<maxSize)": {
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				},
				maxSize: 100,
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				}),
			},
		},
		"success: small message compressed without chunking (input=maxSize)": {
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				},
				maxSize: 58, // size of compressed message
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				}),
			},
		},
		"success: large message compressed with chunking (input>maxSize)": {
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				},
				maxSize: 57, // < size of compressed message
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				})[:57],
				compressOnly(t, &types.Message{
					Type: generated.PingServiceV1Response,
					Content: &pingv1.PingResponse{
						PingMessage: "The quick brown fox jumps over the lazy dog",
					},
				})[57:],
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			c := &chunkingCompressor{tt.args.maxSize}
			got, err := c.Compress(tt.args.msg)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

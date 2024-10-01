/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"testing"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
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
		"err: unknown message type": {
			args: args{msg: &types.Message{Type: "Unknown"}, maxSize: 5},
			err:  types.ErrUnknownMessageType, // TODO@ probably doesn't work. this err is ONLY used there
		},
		"err: empty message": {
			args: args{msg: &types.Message{Type: types.ActivitySearchResponse}, maxSize: 5},
			err:  ErrCompressionProducedNoChunks,
		},
		"success: small message compressed without chunking (input<maxSize)": {
			args: args{
				msg: &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				},
				maxSize: 100,
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				}),
			},
		},
		"success: small message compressed without chunking (input=maxSize)": {
			args: args{
				msg: &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				},
				maxSize: 23, // compressed size of msgType=messages.ActivitySearchResponse and serviceCode="test"
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				}),
			},
		},
		"success: large message compressed with chunking (input>maxSize)": {
			args: args{
				msg: &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				},
				maxSize: 22, // < 23 = compressed size of msgType=messages.ActivitySearchResponse and serviceCode="test"
			},
			want: [][]byte{
				compressOnly(t, &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				})[:22],
				compressOnly(t, &types.Message{
					Type: types.ActivitySearchResponse,
					Content: &activityv2.ActivitySearchResponse{
						Results: []*activityv2.ActivitySearchResult{
							{Info: &activityv2.Activity{ServiceCode: "test"}},
						},
					},
				})[22:],
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

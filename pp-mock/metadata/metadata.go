// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"context"
	"log"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
)

type keyType string // private type to avoid collisions

const key keyType = "metadata"

func ContextWithMetadata(ctx context.Context) context.Context {
	return context.WithValue(ctx, key, metadata.FromGRPCContext(ctx))
}

func FromContext(ctx context.Context) *metadata.Metadata {
	md, ok := ctx.Value(key).(*metadata.Metadata)
	if !ok {
		log.Printf("failed to extract metadata from context")
		md = &metadata.Metadata{}
	}
	return md
}

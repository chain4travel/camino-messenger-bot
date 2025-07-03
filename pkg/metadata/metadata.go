// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/metadata"
)

const (
	KeyRequestID          = "request_id"
	KeyRecipientCMAccount = "recipient_cm_account"
	KeySenderCMAccount    = "sender_cm_account"
	KeyTimestamps         = "timestamps"
)

type Metadata struct {
	RequestID          string
	RecipientCMAccount common.Address
	SenderCMAccount    common.Address
	Timestamps         Timestamps // can be nil
}

func FromGRPCContext(ctx context.Context) *Metadata {
	md := &Metadata{}

	mdPairs, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return md
	}

	if requestID := mdPairs["request_id"]; len(requestID) == 1 {
		md.RequestID = requestID[0]
	}

	if recipientCMAccount := mdPairs["recipient_cm_account"]; len(recipientCMAccount) == 1 {
		md.RecipientCMAccount = common.HexToAddress(recipientCMAccount[0])
	}

	if senderCMAccount := mdPairs["sender_cm_account"]; len(senderCMAccount) == 1 {
		md.SenderCMAccount = common.HexToAddress(senderCMAccount[0])
	}

	if timestampsStr := mdPairs["timestamps"]; len(timestampsStr) == 1 {
		md.Timestamps, _ = TimestampsFromString(timestampsStr[0])
	}

	return md
}

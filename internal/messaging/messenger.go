// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
)

type EncodedSignedMessageWithSender struct {
	Message                EncodedSignedMessage
	SenderBotAddress       common.Address
	SenderCMAccountAddress common.Address
}

type EncodedSignedMessage struct {
	ChunkedEncodedMessage [][]byte
	Signature             []byte
}

type Messenger interface {
	// Initializes messenger and starts receiving messages.
	Start(ctx context.Context) (chan error, error)

	// Stop receiving messages.
	Stop() error

	// Should only be called after Start() was called.
	SendMessage(ctx context.Context, msg *EncodedSignedMessage, sendTo common.Address, networkFeeCheque *cheques.SignedCheque) error

	// Channel where incoming messages are written
	ReceivedMessageChan() chan EncodedSignedMessageWithSender
}

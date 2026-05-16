// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package encoding

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/encryption"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/message"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/metadata"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dummySession struct{}

func (d *dummySession) Commit() error {
	return nil
}

func (d *dummySession) Abort() error {
	return nil
}

func TestEncodeDecodeV1(t *testing.T) {
	ctx := context.Background()
	storageSession := &dummySession{}

	// sender

	senderBotKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)
	senderBotAddress := crypto.PubkeyToAddress(senderBotKey.PublicKey)

	senderStorage := NewMockStorage(gomock.NewController(t))
	senderEncoderDecoder, err := NewEncoderDecoder(
		zap.NewNop().Sugar(),
		senderStorage,
		100,          // max chunk size
		senderBotKey, // key,
	)
	require.NoError(t, err)

	// recipient

	recipientBotKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)
	recipientBotAddress := crypto.PubkeyToAddress(recipientBotKey.PublicKey)

	recipientStorage := NewMockStorage(gomock.NewController(t))
	recipientEncoderDecoder, err := NewEncoderDecoder(
		zap.NewNop().Sugar(),
		recipientStorage,
		100,             // max chunk size
		recipientBotKey, // key,
	)
	require.NoError(t, err)

	// messages

	sharedKey, err := encryption.NewKey()
	require.NoError(t, err)

	requestID := "request-id"

	serviceFeeCheque := &cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: common.Address{4},
			ToCMAccount:   common.Address{5},
			ToBot:         recipientBotAddress, // we need correct address here
			Counter:       big.NewInt(321),
			Amount:        big.NewInt(987),
			CreatedAt:     big.NewInt(3030),
			ExpiresAt:     big.NewInt(4040),
		},
		Signature: []byte{6, 7, 8, 9, 10},
	}

	requestMessage := &message.Message{
		Type: generated.PingServiceV1Request,
		Content: &pingv1.PingRequest{
			PingMessage: "ping",
			Timestamp:   timestamppb.Now(),
		},
		RequestID:  requestID,
		Timestamps: metadata.Timestamps{"1": 100, "2": 200},
	}

	responseMessage := &message.Message{
		Type: generated.PingServiceV1Response,
		Content: &pingv1.PingResponse{
			PingMessage: "pong",
			Timestamp:   timestamppb.Now(),
		},
		RequestID:  requestID,
		Timestamps: metadata.Timestamps{"1": 100, "2": 200, "3": 300},
	}

	// sender encode request

	senderStorage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	senderStorage.EXPECT().GetBotPubKey(ctx, storageSession, recipientBotAddress).Return(&recipientBotKey.PublicKey, nil)
	senderStorage.EXPECT().Abort(storageSession)

	encodedMessage, err := senderEncoderDecoder.EncodeMessage(
		ctx,
		requestMessage,
		serviceFeeCheque,
		recipientBotAddress,
		sharedKey,
	)
	require.NoError(t, err)

	// recipient decode request

	recipientStorage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	recipientStorage.EXPECT().SetBotPubKey(ctx, storageSession, senderBotAddress, &senderBotKey.PublicKey).Return(nil) // also adds to cache
	recipientStorage.EXPECT().Commit(storageSession).Return(nil)
	recipientStorage.EXPECT().Abort(storageSession)

	decodedMessage, decodedServiceFeeCheque, sharedKey, err := recipientEncoderDecoder.DecodeAndVerifyMessage( // we can't verify shared key here, because it is a new key
		ctx,
		encodedMessage,
		senderBotAddress,
	)
	require.NoError(t, err)
	require.True(t, proto.Equal(requestMessage.Content, decodedMessage.Content))
	proto.Reset(requestMessage.Content)
	proto.Reset(decodedMessage.Content)
	require.Equal(t, requestMessage, decodedMessage)
	require.Equal(t, serviceFeeCheque, decodedServiceFeeCheque)

	// recipient encode response (shared key from received request)

	// getting senderBot pubKey from cache, no need to call storage

	encodedMessage, err = recipientEncoderDecoder.EncodeMessage(
		ctx,
		responseMessage,
		nil,
		senderBotAddress,
		sharedKey,
	)
	require.NoError(t, err)

	// sender decode response

	senderStorage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	senderStorage.EXPECT().SetBotPubKey(ctx, storageSession, recipientBotAddress, &recipientBotKey.PublicKey).Return(nil) // also adds to cache
	senderStorage.EXPECT().Commit(storageSession).Return(nil)
	senderStorage.EXPECT().Abort(storageSession)

	decodedMessage, decodedServiceFeeCheque, decodedSharedKey, err := senderEncoderDecoder.DecodeAndVerifyMessage(
		ctx,
		encodedMessage,
		recipientBotAddress,
	)
	require.NoError(t, err)
	require.True(t, proto.Equal(responseMessage.Content, decodedMessage.Content))
	proto.Reset(responseMessage.Content)
	proto.Reset(decodedMessage.Content)
	require.Equal(t, responseMessage, decodedMessage)
	require.Equal(t, sharedKey, decodedSharedKey)
	require.Nil(t, decodedServiceFeeCheque) // we don't have service fee cheque in response messages
}

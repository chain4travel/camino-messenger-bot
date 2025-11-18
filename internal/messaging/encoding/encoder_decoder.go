// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package encoding

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encryption"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/metadata"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var _ messaging.EncoderDecoder = (*encoderDecoder)(nil)

const (
	pubKeysCacheSize = 100

	messageVersion        uint32 = 1
	defaultExpirationTime        = 5 * time.Minute

	messageVersionSize     = 4 // uint32 size
	publicMetadataLenSize  = 2 // uint16 size
	privateMetadataLenSize = 2 // uint16 size
)

func NewEncoderDecoder(
	logger *zap.SugaredLogger,
	storage Storage,
	maxChunkSize int,
	key *ecdsa.PrivateKey,
) (messaging.EncoderDecoder, error) {
	zstdEncoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, err
	}
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	pubKeysCache, err := lru.New[common.Address, *ecdsa.PublicKey](pubKeysCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create public keys cache: %w", err)
	}

	return &encoderDecoder{
		storage:      storage,
		ecdsaPrivKey: key,
		eciesPubKey:  ecies.ImportECDSAPublic(&key.PublicKey),
		eciesPrivKey: ecies.ImportECDSA(key),
		logger:       logger,
		zstdEncoder:  zstdEncoder,
		zstdDecoder:  zstdDecoder,
		maxChunkSize: maxChunkSize,
		pubKeysCache: pubKeysCache,
	}, nil
}

type encoderDecoder struct {
	ecdsaPrivKey *ecdsa.PrivateKey
	eciesPubKey  *ecies.PublicKey
	eciesPrivKey *ecies.PrivateKey
	maxChunkSize int

	logger       *zap.SugaredLogger
	zstdEncoder  *zstd.Encoder
	zstdDecoder  *zstd.Decoder
	storage      Storage
	pubKeysCache *lru.Cache[common.Address, *ecdsa.PublicKey]
}

type publicMetadata struct {
	RequestID          string `json:"request_id"`
	EncryptedSharedKey []byte `json:"encrypted_shared_key,omitempty"`
	ExpiresAt          uint64 `json:"expires_at"` // Unix timestamp in seconds, when the message is no longer valid
}

func (m *publicMetadata) ExpiresAtTime() time.Time {
	return time.Unix(conversion.MustUInt64ToInt64(m.ExpiresAt), 0)
}

type privateMetadata struct {
	Type             types.MessageType     `json:"type"`
	Timestamps       metadata.Timestamps   `json:"timestamps"`
	ServiceFeeCheque *cheques.SignedCheque `json:"service_fee_cheque,omitempty"`
}

func (e *encoderDecoder) EncodeMessage(
	ctx context.Context,
	msg *types.Message,
	serviceFeeCheque *cheques.SignedCheque,
	toBot common.Address,
	sharedKey encryption.Key,
) (*messaging.EncodedSignedMessage, error) {
	recipientECDSAPubKey, err := e.getBotPubKey(ctx, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient public key: %w", err)
	}

	var encryptedSharedKey []byte
	if recipientECDSAPubKey != nil && sharedKey != nil {
		encryptedSharedKey, err = ecies.Encrypt(
			rand.Reader,
			ecies.ImportECDSAPublic(recipientECDSAPubKey),
			sharedKey.Bytes(),
			nil,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt session key: %w", err)
		}
	}

	privateMetadata, err := json.Marshal(&privateMetadata{
		ServiceFeeCheque: serviceFeeCheque,
		Type:             msg.Type,
		Timestamps:       msg.Timestamps,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private metadata: %w", err)
	}
	privateMetadataLen := len(privateMetadata)

	publicMetadata, err := json.Marshal(&publicMetadata{
		RequestID:          msg.RequestID,
		EncryptedSharedKey: encryptedSharedKey,
		ExpiresAt:          conversion.MustInt64ToUInt64(time.Now().Add(defaultExpirationTime).Unix()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public metadata: %w", err)
	}
	publicMetadataLen := len(publicMetadata)

	content, err := proto.Marshal(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message content: %w", err)
	}

	// privateData = len(PrivateMetadata) + PrivateMetadata + content
	privateData := make([]byte, privateMetadataLenSize+privateMetadataLen+len(content))
	binary.BigEndian.PutUint16(privateData[:privateMetadataLenSize], conversion.MustIntToUInt16(privateMetadataLen))
	copy(privateData[privateMetadataLenSize:], privateMetadata)
	copy(privateData[privateMetadataLenSize+privateMetadataLen:], content)

	encryptedData := e.compress(privateData)
	if encryptedSharedKey != nil {
		encryptedData, err = sharedKey.Encrypt(encryptedData)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt data with shared key: %w", err)
		}
	}

	// encodedData = messageVersion + len(publicMetadata) + publicMetadata + encryptedData
	encodedData := make([]byte, messageVersionSize+publicMetadataLenSize+publicMetadataLen+len(encryptedData))
	binary.BigEndian.PutUint32(encodedData[:messageVersionSize], messageVersion)
	binary.BigEndian.PutUint16(encodedData[messageVersionSize:], conversion.MustIntToUInt16(publicMetadataLen))
	copy(encodedData[messageVersionSize+publicMetadataLenSize:], publicMetadata)
	copy(encodedData[messageVersionSize+publicMetadataLenSize+publicMetadataLen:], encryptedData)

	signature, err := crypto.Sign(crypto.Keccak256(encodedData), e.ecdsaPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return &messaging.EncodedSignedMessage{
		ChunkedEncodedMessage: e.chunk(encodedData),
		Signature:             signature,
	}, nil
}

func (e *encoderDecoder) DecodeAndVerifyMessage(
	ctx context.Context,
	encodedMessage *messaging.EncodedSignedMessage,
	senderBotAddress common.Address,
) (
	msg *types.Message,
	serviceFeeCheque *cheques.SignedCheque,
	sharedKey encryption.Key,
	err error,
) {
	encodedData := bytes.Join(encodedMessage.ChunkedEncodedMessage, nil)

	version := binary.BigEndian.Uint32(encodedData[:messageVersionSize])
	if version != messageVersion {
		return nil, nil, nil, fmt.Errorf("unsupported message version: %d", version)
	}

	senderPubKey, err := crypto.SigToPub(crypto.Keccak256(encodedData), encodedMessage.Signature)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to recover public key from signature: %w", err)
	}
	if crypto.PubkeyToAddress(*senderPubKey) != senderBotAddress {
		return nil, nil, nil, fmt.Errorf("sender address %s does not match recovered address %s", senderBotAddress.Hex(), crypto.PubkeyToAddress(*senderPubKey).Hex())
	}

	msg, serviceFeeCheque, encryptionKey, err := e.decodeAndVerifyMessageV1(encodedData[messageVersionSize:])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode and verify message: %w", err)
	}

	if err := e.setBotPubKey(ctx, senderBotAddress, senderPubKey); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to store sender public key: %w", err)
	}

	if msg.Timestamps == nil {
		msg.Timestamps = metadata.Timestamps{}
	}

	return msg, serviceFeeCheque, encryptionKey, nil
}

func (e *encoderDecoder) decodeAndVerifyMessageV1(
	encodedData []byte,
) (
	msg *types.Message,
	serviceFeeCheque *cheques.SignedCheque,
	sharedKey encryption.Key,
	err error,
) {
	publicMetadataLen := binary.BigEndian.Uint16(encodedData[:publicMetadataLenSize])
	publicMetadata := &publicMetadata{}
	if err := json.Unmarshal(encodedData[publicMetadataLenSize:publicMetadataLenSize+publicMetadataLen], publicMetadata); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if publicMetadata.ExpiresAtTime().Before(time.Now()) {
		return nil, nil, nil, fmt.Errorf("message has expired at %s", publicMetadata.ExpiresAtTime().Format(time.RFC3339))
	}

	compressedData := encodedData[publicMetadataLenSize+publicMetadataLen:]
	if len(publicMetadata.EncryptedSharedKey) > 0 {
		sharedKeyBytes, err := e.eciesPrivKey.Decrypt(publicMetadata.EncryptedSharedKey, nil, nil)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to decrypt shared key: %w", err)
		}

		sharedKey, err = encryption.KeyFromBytes(sharedKeyBytes)
		if err != nil {
			return nil, nil, nil, err
		}

		compressedData, err = sharedKey.Decrypt(compressedData)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to decrypt data with shared key: %w", err)
		}
	}

	privateData, err := e.decompress(compressedData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	privateMetadataLen := binary.BigEndian.Uint16(privateData[:privateMetadataLenSize])
	privateMetadata := &privateMetadata{}
	if err := json.Unmarshal(privateData[privateMetadataLenSize:privateMetadataLenSize+privateMetadataLen], &privateMetadata); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal private metadata: %w", err)
	}

	msg = &types.Message{
		RequestID:  publicMetadata.RequestID,
		Type:       privateMetadata.Type,
		Timestamps: privateMetadata.Timestamps,
	}

	if err := generated.UnmarshalContent(privateData[privateMetadataLenSize+privateMetadataLen:], msg.Type, &msg.Content); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal content: %w", err)
	}

	return msg, privateMetadata.ServiceFeeCheque, sharedKey, nil
}

func (e *encoderDecoder) getBotPubKey(
	ctx context.Context,
	botAddress common.Address,
) (*ecdsa.PublicKey, error) {
	if pubKey, ok := e.pubKeysCache.Get(botAddress); ok {
		return pubKey, nil
	}

	session, err := e.storage.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}
	defer e.storage.Abort(session)

	pubKey, err := e.storage.GetBotPubKey(ctx, session, botAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot public key: %w", err)
	}

	e.pubKeysCache.Add(botAddress, pubKey)

	return pubKey, nil
}

func (e *encoderDecoder) setBotPubKey(
	ctx context.Context,
	botAddress common.Address,
	botPubKey *ecdsa.PublicKey,
) error {
	session, err := e.storage.NewSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new session: %w", err)
	}
	defer e.storage.Abort(session)

	if err := e.storage.SetBotPubKey(ctx, session, botAddress, botPubKey); err != nil {
		return fmt.Errorf("failed to store bot public key: %w", err)
	}

	if err := e.storage.Commit(session); err != nil {
		return fmt.Errorf("failed to commit session: %w", err)
	}

	_ = e.pubKeysCache.Add(botAddress, botPubKey)

	return nil
}

func (e *encoderDecoder) compress(src []byte) []byte {
	return e.zstdEncoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func (e *encoderDecoder) decompress(src []byte) ([]byte, error) {
	return e.zstdDecoder.DecodeAll(src, nil)
}

func (e *encoderDecoder) chunk(data []byte) [][]byte {
	numChunks := (len(data) + e.maxChunkSize - 1) / e.maxChunkSize
	result := make([][]byte, numChunks)
	start := 0

	for i := range numChunks {
		end := min(start+e.maxChunkSize, len(data))
		result[i] = make([]byte, end-start)
		copy(result[i], data[start:end])
		start = end
	}

	return result
}

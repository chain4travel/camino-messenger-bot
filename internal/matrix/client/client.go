// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/matrix/messenger"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/matrix"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	// required to initialize the sqlite driver
	_ "github.com/mattn/go-sqlite3"
)

const (
	// MaxChunkSize a moderate/safe max chunk size is 48KB. This is because the maximum size of a matrix event is 64KB.
	// Megolm encryption adds an extra 33% overhead to the encrypted content due to base64 encryption. This means that
	// the maximum size of pre-encrypted chunk should be 48KB / 1.33 ~= 36KB. We round down to 35KB to be safe.
	MaxChunkSize = 30 << 10 // max pre-encrypted chunk size is 30KB - 35KB proved to be an unsafe limit (TODO investigate optimal limit)
)

var (
	_ messenger.Client = (*client)(nil)

	errNotDefaultSyncer = errors.New("matrix client syncer is not of type DefaultSyncer")
)

func New(
	ctx context.Context,
	logger *zap.SugaredLogger,
	homeserverURL string,
	dbPath string,
	botKey *ecdsa.PrivateKey,
	expectedBotUserID id.UserID,
) (messenger.Client, error) {
	matrixClient, err := mautrix.NewClient(homeserverURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to create matrix client: %w", err)
	}

	syncer, ok := matrixClient.Syncer.(*mautrix.DefaultSyncer)
	if !ok { // should never happen with current mautrix implementation
		return nil, errNotDefaultSyncer
	}

	cryptoHelper, err := cryptohelper.NewCryptoHelper(matrixClient, []byte("meow"), dbPath) // TODO @nikos refactor
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto helper: %w", err)
	}

	signature, message, err := SignPublicKey(botKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign public key: %w", err)
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	c := &client{
		logger:       logger,
		client:       matrixClient,
		syncer:       syncer,
		cryptoHelper: cryptoHelper,
	}

	if err := c.initCryptoHelper(ctx, expectedBotUserID); err != nil {
		if closeErr := c.cryptoHelper.Close(); closeErr != nil {
			logger.Errorf("failed to close crypto helper after crypto helper init failure: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to initialize crypto helper: %w", err)
	}

	logger.Infof("Successfully logged in as: %s", c.client.UserID)

	return c, nil
}

type client struct {
	logger       *zap.SugaredLogger
	client       *mautrix.Client
	syncer       *mautrix.DefaultSyncer
	cryptoHelper *cryptohelper.CryptoHelper
}

func (c *client) initCryptoHelper(ctx context.Context, expectedBotUserID id.UserID) error {
	if err := c.cryptoHelper.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize crypto helper: %w", err)
	}

	if c.client.UserID != expectedBotUserID {
		return fmt.Errorf("logged in user ID %s does not match expected bot user ID %s", c.client.UserID, expectedBotUserID)
	}

	c.client.Crypto = c.cryptoHelper

	return nil
}

func (c *client) Close() error {
	return c.cryptoHelper.Close()
}

func (c *client) SetEventHandler(eventType event.Type, handler func(ctx context.Context, evt *event.Event)) {
	c.syncer.OnEventType(eventType, handler)
}

func (c *client) IsRoomEncrypted(ctx context.Context, roomID id.RoomID) (bool, error) {
	return c.client.StateStore.IsEncrypted(ctx, roomID)
}

func (c *client) CreateRoomForUser(ctx context.Context, recipient id.UserID) (id.RoomID, error) {
	c.logger.Debugf("Creating new room for user %v", recipient)
	resp, err := c.client.CreateRoom(ctx, &mautrix.ReqCreateRoom{
		Visibility: "private",
		Preset:     "private_chat",
		Invite:     []id.UserID{recipient},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create new room for user %s: %w", recipient, err)
	}
	c.logger.Debugf("Created new room %s for user %s", resp.RoomID, recipient)
	return resp.RoomID, nil
}

func (c *client) JoinRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Joining room %s", roomID)
	if _, err := c.client.JoinRoomByID(ctx, roomID); err != nil {
		return fmt.Errorf("failed to join room %s: %w", roomID, err)
	}
	c.logger.Debugf("Joined room %s", roomID)
	return nil
}

func (c *client) LeaveRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Leaving room %s", roomID)
	if _, err := c.client.LeaveRoom(ctx, roomID); err != nil {
		return fmt.Errorf("failed to leave room %s: %w", roomID, err)
	}
	c.logger.Debugf("Left room %s", roomID)
	return nil
}

func (c *client) ForgetRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Forgetting room %s", roomID)
	if _, err := c.client.ForgetRoom(ctx, roomID); err != nil {
		return fmt.Errorf("failed to forget room %s: %w", roomID, err)
	}
	c.logger.Debugf("Forgot room %s", roomID)
	return nil
}

func (c *client) JoinedRooms(ctx context.Context) ([]id.RoomID, error) {
	resp, err := c.client.JoinedRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get joined rooms: %w", err)
	}
	return resp.JoinedRooms, nil
}

func (c *client) IsUserJoinedRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) (bool, error) {
	resp, err := c.client.JoinedMembers(ctx, roomID)
	if err != nil {
		return false, fmt.Errorf("failed to get joined members for room %s: %w", roomID, err)
	}
	_, joined := resp.Joined[userID]
	return joined, nil
}

func (c *client) SendSignedMessageEvent(ctx context.Context, roomID id.RoomID, event matrix.SignedMessageEventContent) error {
	c.logger.Debugf("Sending message event of type %s to room %s", matrix.EventTypeSignedMessage.Type, roomID)
	_, err := c.client.SendMessageEvent(ctx, roomID, matrix.EventTypeSignedMessage, event, mautrix.ReqSendEvent{DontEncrypt: true})
	if err != nil {
		return fmt.Errorf("failed to send message event of type %s to room %s: %w", matrix.EventTypeSignedMessage.Type, roomID, err)
	}
	c.logger.Debugf("Sent message event of type %s to room %s", matrix.EventTypeSignedMessage.Type, roomID)
	return nil
}

func (c *client) SendMessageChunkEvent(ctx context.Context, roomID id.RoomID, event matrix.MessageChunkEventContent) error {
	c.logger.Debugf("Sending message event of type %s to room %s", matrix.EventTypeMessageChunk.Type, roomID)
	_, err := c.client.SendMessageEvent(ctx, roomID, matrix.EventTypeMessageChunk, event, mautrix.ReqSendEvent{DontEncrypt: true})
	if err != nil {
		return fmt.Errorf("failed to send message event of type %s to room %s: %w", matrix.EventTypeMessageChunk.Type, roomID, err)
	}
	c.logger.Debugf("Sent message event of type %s to room %s", matrix.EventTypeMessageChunk.Type, roomID)
	return nil
}

func (c *client) SyncWithContext(ctx context.Context) error {
	c.logger.Debug("Starting sync")
	if err := c.client.SyncWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("sync failed: %w", err)
	}
	c.logger.Debug("Sync finished")
	return nil
}

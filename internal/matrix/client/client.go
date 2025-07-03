// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/matrix/messenger"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	// required to initialize the sqlite driver
	_ "github.com/mattn/go-sqlite3"
)

var _ messenger.Client = (*client)(nil)

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
		logger.Errorf("failed to create matrix client: %v", err)
		return nil, err
	}

	syncer, ok := matrixClient.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		return nil, fmt.Errorf("failed to cast syncer to DefaultSyncer")
	}

	cryptoHelper, err := cryptohelper.NewCryptoHelper(matrixClient, []byte("meow"), dbPath) // TODO @nikos refactor
	if err != nil {
		return nil, err
	}

	signature, message, err := SignPublicKey(botKey)
	if err != nil {
		return nil, err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	if err = cryptoHelper.Init(ctx); err != nil {
		_ = cryptoHelper.Close()
		return nil, err
	}

	if matrixClient.UserID != expectedBotUserID {
		_ = cryptoHelper.Close()
		return nil, fmt.Errorf("expected user ID %s, got %s", expectedBotUserID, matrixClient.UserID)
	}

	matrixClient.Crypto = cryptoHelper

	logger.Infof("Successfully logged in as: %s", matrixClient.UserID)

	return &client{
		logger:       logger,
		client:       matrixClient,
		syncer:       syncer,
		cryptoHelper: cryptoHelper,
	}, nil
}

type client struct {
	logger       *zap.SugaredLogger
	client       *mautrix.Client
	syncer       *mautrix.DefaultSyncer
	cryptoHelper *cryptohelper.CryptoHelper
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
		c.logger.Errorf("Failed to create new room for user %s: %v", recipient, err)
		return "", err
	}
	c.logger.Debugf("Created new room %s for user %s", resp.RoomID, recipient)
	return resp.RoomID, nil
}

func (c *client) JoinRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Joining room %s", roomID)
	if _, err := c.client.JoinRoomByID(ctx, roomID); err != nil {
		c.logger.Errorf("Failed to join room %s: %v", roomID, err)
		return err
	}
	c.logger.Debugf("Joined room %s", roomID)
	return nil
}

func (c *client) LeaveRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Leaving room %s", roomID)
	if _, err := c.client.LeaveRoom(ctx, roomID); err != nil {
		c.logger.Errorf("Failed to leave room %s: %v", roomID, err)
		return err
	}
	c.logger.Debugf("Left room %s", roomID)
	return nil
}

func (c *client) ForgetRoom(ctx context.Context, roomID id.RoomID) error {
	c.logger.Debugf("Forgetting room %s", roomID)
	if _, err := c.client.ForgetRoom(ctx, roomID); err != nil {
		c.logger.Errorf("Failed to forget room %s: %v", roomID, err)
		return err
	}
	c.logger.Debugf("Forgot room %s", roomID)
	return nil
}

func (c *client) JoinedRooms(ctx context.Context) ([]id.RoomID, error) {
	resp, err := c.client.JoinedRooms(ctx)
	if err != nil {
		c.logger.Errorf("Failed to get joined rooms: %v", err)
		return nil, err
	}
	return resp.JoinedRooms, nil
}

func (c *client) IsUserJoinedRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) (bool, error) {
	resp, err := c.client.JoinedMembers(ctx, roomID)
	if err != nil {
		c.logger.Errorf("Failed to get joined members for room %s: %v", roomID, err)
		return false, err
	}
	_, joined := resp.Joined[userID]
	return joined, nil
}

func (c *client) SendMessageEvent(ctx context.Context, roomID id.RoomID, event matrix.MessageEventContent) error {
	c.logger.Debugf("Sending message event of type %s to room %s", matrix.EventTypeMessage.Type, roomID)
	_, err := c.client.SendMessageEvent(ctx, roomID, matrix.EventTypeMessage, event)
	if err != nil {
		c.logger.Errorf("Failed to send message event of type %s to room %s: %v", matrix.EventTypeMessage.Type, roomID, err)
		return err
	}
	c.logger.Debugf("Sent message event of type %s to room %s", matrix.EventTypeMessage.Type, roomID)
	return nil
}

func (c *client) SendMessageChunkEvent(ctx context.Context, roomID id.RoomID, event matrix.MessageChunkEventContent) error {
	c.logger.Debugf("Sending message event of type %s to room %s", matrix.EventTypeMessageChunk.Type, roomID)
	_, err := c.client.SendMessageEvent(ctx, roomID, matrix.EventTypeMessageChunk, event)
	if err != nil {
		c.logger.Errorf("Failed to send message event of type %s to room %s: %v", matrix.EventTypeMessageChunk.Type, roomID, err)
		return err
	}
	c.logger.Debugf("Sent message event of type %s to room %s", matrix.EventTypeMessageChunk.Type, roomID)
	return nil
}

func (c *client) SyncWithContext(ctx context.Context) error {
	c.logger.Debug("Starting sync")
	if err := c.client.SyncWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		c.logger.Errorf("Sync failed: %v", err)
		return err
	}
	c.logger.Debug("Sync finished")
	return nil
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/matrix"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const roomsCacheSize = 100

var _ messaging.Messenger = (*messenger)(nil)

func NewMessenger(
	logger *zap.SugaredLogger,
	matrixClient Client,
	botKey *ecdsa.PrivateKey,
	botUserID id.UserID,
) (messaging.Messenger, error) {
	roomsCache, err := lru.New[id.UserID, id.RoomID](roomsCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create rooms cache: %w", err)
	}

	m := &messenger{
		msgChannel:      make(chan messaging.EncodedSignedMessageWithSender),
		logger:          logger,
		client:          matrixClient,
		rooms:           roomsCache,
		chunkedMessages: make(map[string]*chunkedMessage),
		botKey:          botKey,
		botUserID:       botUserID,
		matrixHost:      botUserID.Homeserver(),
	}

	m.client.SetEventHandler(matrix.EventTypeSignedMessage, m.signedMessageEventHandler)
	m.client.SetEventHandler(matrix.EventTypeMessageChunk, m.messageChunkEventHandler)
	m.client.SetEventHandler(event.StateMember, m.stateMemberEventHandler)

	return m, nil
}

type messenger struct {
	botKey     *ecdsa.PrivateKey
	botUserID  id.UserID
	matrixHost string

	msgChannel      chan messaging.EncodedSignedMessageWithSender
	rooms           *lru.Cache[id.UserID, id.RoomID]
	chunkedMessages map[string]*chunkedMessage
	messagesMutex   sync.RWMutex
	cancelSync      func()
	syncerDoneChan  chan struct{}

	logger *zap.SugaredLogger
	client Client
}

func (m *messenger) ReceivedMessageChan() chan messaging.EncodedSignedMessageWithSender {
	return m.msgChannel
}

func (m *messenger) Start(ctx context.Context) (chan error, error) {
	// Initially, messenger used matrix e2e encryption. Starting from v12, messenger will use its own e2e encryption.
	// Because of that, we need to remove all encrypted rooms and only use unencrypted rooms from now on.
	if err := m.removeEncryptedRooms(ctx); err != nil {
		err := fmt.Errorf("failed to remove encrypted rooms: %w", err)
		m.logger.Error(err)
		return nil, err
	}

	// Start syncer, that will listen for incoming events

	syncCtx, cancelSync := context.WithCancel(ctx)
	errChan := make(chan error)
	m.cancelSync = cancelSync
	m.syncerDoneChan = make(chan struct{})

	// by default, the syncer will get all events from the last hour
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("matrix event syncer panicked: %v", r)
				m.logger.Errorf("recovered from panic: %v", err)
				errChan <- err
			}
			close(errChan)
			close(m.syncerDoneChan)
		}()

		if err := m.client.SyncWithContext(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
			err := fmt.Errorf("matrix event syncer exited with error: %w", err)
			m.logger.Error(err)
			errChan <- err
		}
	}()

	return errChan, nil
}

func (m *messenger) Stop() error {
	if m.cancelSync != nil {
		m.logger.Info("Stopping matrix syncer...")
		// if cancelSync is not nil, it means that the syncer is running
		m.cancelSync()
		// we only wait for the syncer to stop if it was running,
		// otherwise no one will close syncerStopChan
		// and the goroutine will block forever
		<-m.syncerDoneChan
		m.logger.Info("Matrix syncer stopped")
	}

	if err := m.client.Close(); err != nil {
		err = fmt.Errorf("failed to close matrix client: %w", err)
		m.logger.Error(err)
		return err
	}

	return nil
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func TestGetRoomForRecipient(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop().Sugar()
	botKey := testKey

	recipientUserID := id.UserID("recipientUserID")
	roomID1 := id.RoomID("roomID1")
	roomID2 := id.RoomID("roomID2")
	zeroRoomID := id.RoomID("")

	testErr := errors.New("test err")

	// context expected in arg 0 is always child context created inside function

	tests := map[string]struct {
		client         func(*gomock.Controller) *MockClient
		roomsCache     map[id.UserID]id.RoomID
		recipient      id.UserID
		expectedRoomID id.RoomID
		expectedErr    error
	}{
		"Create room fails": {
			client: func(ctrl *gomock.Controller) *MockClient {
				c := NewMockClient(ctrl)
				c.EXPECT().JoinedRooms(ctx).Return([]id.RoomID{}, nil)
				c.EXPECT().CreateRoomForUser(ctx, recipientUserID).Return(zeroRoomID, testErr)
				return c
			},
			recipient:   recipientUserID,
			expectedErr: testErr,
		},
		"OK: room already established and cached": {
			roomsCache: map[id.UserID]id.RoomID{
				recipientUserID: roomID1,
			},
			recipient:      recipientUserID,
			expectedRoomID: roomID1,
		},
		"OK: room already established but not cached": {
			client: func(ctrl *gomock.Controller) *MockClient {
				c := NewMockClient(ctrl)
				c.EXPECT().JoinedRooms(ctx).Return([]id.RoomID{roomID1, roomID2}, nil)
				c.EXPECT().IsUserJoinedRoom(ctx, roomID1, recipientUserID).Return(false, nil)
				c.EXPECT().IsUserJoinedRoom(ctx, roomID2, recipientUserID).Return(true, nil)
				return c
			},
			recipient:      recipientUserID,
			expectedRoomID: roomID2,
		},
		"OK: room exists but recipient is not member so create new encrypted room created and invite user": {
			client: func(ctrl *gomock.Controller) *MockClient {
				c := NewMockClient(ctrl)
				c.EXPECT().JoinedRooms(ctx).Return([]id.RoomID{roomID1}, nil)
				c.EXPECT().IsUserJoinedRoom(ctx, roomID1, recipientUserID).Return(false, nil)
				c.EXPECT().CreateRoomForUser(ctx, recipientUserID).Return(roomID2, nil)
				return c
			},
			recipient:      recipientUserID,
			expectedRoomID: roomID2,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			if tt.client == nil {
				tt.client = NewMockClient
			}
			matrixClient := tt.client(ctrl)
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeMessageChunk, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(matrix.EventTypeSignedMessage, gomock.Any())
			matrixClient.EXPECT().SetEventHandler(event.StateMember, gomock.Any())

			matrixMessenger, err := NewMessenger(
				logger,
				matrixClient,
				botKey,
				id.UserID("botUserID"),
			)
			require.NoError(t, err)

			matrixMessengerImpl := matrixMessenger.(*messenger)
			for userID, roomID := range tt.roomsCache {
				matrixMessengerImpl.rooms.Add(userID, roomID)
			}

			roomID, err := matrixMessengerImpl.getRoomForRecipient(ctx, tt.recipient)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, tt.expectedRoomID, roomID)
		})
	}
}

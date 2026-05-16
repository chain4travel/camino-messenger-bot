// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/matrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Client interface {
	SetEventHandler(eventType event.Type, handler func(ctx context.Context, evt *event.Event))
	IsRoomEncrypted(ctx context.Context, roomID id.RoomID) (bool, error)
	CreateRoomForUser(ctx context.Context, recipient id.UserID) (id.RoomID, error)
	JoinRoom(ctx context.Context, roomID id.RoomID) error
	LeaveRoom(ctx context.Context, roomID id.RoomID) error
	ForgetRoom(ctx context.Context, roomID id.RoomID) error
	JoinedRooms(ctx context.Context) ([]id.RoomID, error)
	IsUserJoinedRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) (bool, error)
	SendSignedMessageEvent(ctx context.Context, roomID id.RoomID, event matrix.SignedMessageEventContent) error
	SendMessageChunkEvent(ctx context.Context, roomID id.RoomID, event matrix.MessageChunkEventContent) error
	SyncWithContext(ctx context.Context) error
	Close() error
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"

	"go.uber.org/zap"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func (m *messenger) stateMemberEventHandler(ctx context.Context, evt *event.Event) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf("failed to process %s event, recovered from panic: %v", event.StateMember.Type, r)
		}
	}()
	m.logger.Debugf("Received %s event %s from %s in room %s", event.StateMember.Type, evt.ID, evt.Sender, evt.RoomID)

	if evt.GetStateKey() == m.botUserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
		if err := m.client.JoinRoom(ctx, evt.RoomID); err != nil {
			m.logger.Error("Failed to join room after invite",
				zap.String("room_id", evt.RoomID.String()),
				zap.String("inviter", evt.Sender.String()))
			return
		}

		m.logger.Info("Joined room after invite",
			zap.String("room_id", evt.RoomID.String()),
			zap.String("inviter", evt.Sender.String()))
	}
}

func (m *messenger) getRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, error) {
	roomID, found := m.findExistingRoomForRecipient(ctx, recipient)
	if found {
		return roomID, nil
	}

	roomID, err := m.client.CreateRoomForUser(ctx, recipient)
	if err != nil {
		return "", err
	}

	if err := m.client.EnableRoomEncryption(ctx, roomID); err != nil {
		m.logger.Errorf("failed to enable encryption for room %s: %v", roomID, err)
		return "", err
	}

	m.rooms.Add(recipient, roomID)

	return roomID, err
}

func (m *messenger) findExistingRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, bool) {
	roomID, ok := m.rooms.Get(recipient)
	if ok {
		return roomID, true
	}

	rooms, err := m.client.JoinedRooms(ctx)
	if err != nil {
		m.logger.Errorf("failed to get joined rooms: %v", err)
		return "", false
	}

	for _, roomID := range rooms {
		if encrypted, err := m.client.IsRoomEncrypted(ctx, roomID); err != nil {
			m.logger.Errorf("failed to check if room %s is encrypted: %v", roomID, err)
			return "", false
		} else if !encrypted {
			continue
		}

		if joined, err := m.client.IsUserJoinedRoom(ctx, roomID, recipient); err != nil {
			m.logger.Errorf("failed to check if user %s is joined to room %s: %v", recipient, roomID, err)
			return "", false
		} else if joined {
			m.rooms.Add(recipient, roomID)
			return roomID, true
		}
	}

	return "", false
}

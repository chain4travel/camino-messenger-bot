/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"errors"
	"testing"

	"maunium.net/go/mautrix/event"

	"maunium.net/go/mautrix"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"maunium.net/go/mautrix/id"
)

func TestGetOrCreateRoomForRecipient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRoomClient := NewMockClient(mockCtrl)
	defer mockCtrl.Finish()

	userID := id.UserID("userID")
	roomID := id.RoomID("roomID")
	newRoomID := id.RoomID("newRoomID")

	errCreateRoomFailed := errors.New("create-room-failed")
	errEnableEncryptionFailed := errors.New("enable-encryption-failed")

	type fields struct {
		rooms map[id.UserID]id.RoomID
	}
	type args struct {
		recipient id.UserID
	}
	tests := map[string]struct {
		fields fields
		args   args
		want   id.RoomID
		mocks  func(r *roomHandler)
		err    error
	}{
		"err: create new encrypted room fails": {
			fields: fields{
				rooms: map[id.UserID]id.RoomID{},
			},
			mocks: func(*roomHandler) {
				mockRoomClient.EXPECT().JoinedRooms().Times(1).Return(&mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{roomID}}, nil)
				mockRoomClient.EXPECT().IsEncrypted(roomID).Times(1).Return(false)
				mockRoomClient.EXPECT().CreateRoom(&mautrix.ReqCreateRoom{
					Visibility: "private",
					Preset:     "private_chat",
					Invite:     []id.UserID{userID},
				}).Times(1).Return(nil, errCreateRoomFailed)
			},
			args: args{recipient: userID},
			err:  errCreateRoomFailed,
		},
		"err: room exists but is unencrypted so create new encrypted room created but enable encryption fails": { //nolint:dupl
			fields: fields{
				rooms: map[id.UserID]id.RoomID{},
			},
			mocks: func(*roomHandler) {
				mockRoomClient.EXPECT().JoinedRooms().Times(1).Return(&mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{roomID}}, nil)
				mockRoomClient.EXPECT().IsEncrypted(roomID).Times(1).Return(false)
				mockRoomClient.EXPECT().CreateRoom(&mautrix.ReqCreateRoom{
					Visibility: "private",
					Preset:     "private_chat",
					Invite:     []id.UserID{userID},
				}).Times(1).Return(&mautrix.RespCreateRoom{RoomID: newRoomID}, nil)
				mockRoomClient.EXPECT().SendStateEvent(newRoomID, event.StateEncryption, "",
					event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1}).Times(1).Return(nil, errEnableEncryptionFailed)
			},
			args: args{recipient: userID},
			err:  errEnableEncryptionFailed,
		},
		"success: room already established and cached": {
			fields: fields{
				rooms: map[id.UserID]id.RoomID{userID: roomID},
			},
			args: args{recipient: userID},
			want: roomID,
		},
		"success: room already established but not cached": {
			fields: fields{
				rooms: map[id.UserID]id.RoomID{},
			},
			mocks: func(*roomHandler) {
				mockRoomClient.EXPECT().JoinedRooms().Times(1).Return(&mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{roomID}}, nil)
				mockRoomClient.EXPECT().IsEncrypted(roomID).Times(1).Return(true)
				mockRoomClient.EXPECT().JoinedMembers(roomID).Times(1).Return(&mautrix.RespJoinedMembers{Joined: map[id.UserID]mautrix.JoinedMember{userID: {}}}, nil)
			},
			args: args{recipient: userID},
			want: roomID,
		},
		"success: room exists but recipient is not member so create new encrypted room created and invite user": { //nolint:dupl
			fields: fields{
				rooms: map[id.UserID]id.RoomID{},
			},
			mocks: func(*roomHandler) {
				mockRoomClient.EXPECT().JoinedRooms().Times(1).Return(&mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{}}, nil)
				mockRoomClient.EXPECT().CreateRoom(&mautrix.ReqCreateRoom{
					Visibility: "private",
					Preset:     "private_chat",
					Invite:     []id.UserID{userID},
				}).Times(1).Return(&mautrix.RespCreateRoom{RoomID: newRoomID}, nil)
				mockRoomClient.EXPECT().SendStateEvent(newRoomID, event.StateEncryption, "",
					event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1}).Times(1).Return(nil, nil)
			},
			args: args{recipient: userID},
			want: newRoomID,
		},
		"success: room exists but is unencrypted so create new encrypted room created and invite user": { //nolint:dupl
			fields: fields{
				rooms: map[id.UserID]id.RoomID{},
			},
			mocks: func(*roomHandler) {
				mockRoomClient.EXPECT().JoinedRooms().Times(1).Return(&mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{roomID}}, nil)
				mockRoomClient.EXPECT().IsEncrypted(roomID).Times(1).Return(false)
				mockRoomClient.EXPECT().CreateRoom(&mautrix.ReqCreateRoom{
					Visibility: "private",
					Preset:     "private_chat",
					Invite:     []id.UserID{userID},
				}).Times(1).Return(&mautrix.RespCreateRoom{RoomID: newRoomID}, nil)
				mockRoomClient.EXPECT().SendStateEvent(newRoomID, event.StateEncryption, "",
					event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1}).Times(1).Return(nil, nil)
			},
			args: args{recipient: userID},
			want: newRoomID,
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			r := &roomHandler{
				client: mockRoomClient,
				logger: zap.NewNop().Sugar(),
				rooms:  tt.fields.rooms,
			}
			if tt.mocks != nil {
				tt.mocks(r)
			}

			got, err := r.GetOrCreateRoomForRecipient(tt.args.recipient)
			require.ErrorIs(t, err, tt.err, "GetOrCreateRoomForRecipient() error = %w, wantErr %w", err, tt.err)
			require.Equal(t, got, tt.want, "GetOrCreateRoomForRecipient() got = %v, expRoomID %v", got, tt.want)
		})
	}
}

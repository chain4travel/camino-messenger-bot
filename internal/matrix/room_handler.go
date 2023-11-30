package matrix

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type RoomHandler interface {
	GetOrCreateRoomForRecipient(context.Context, id.UserID) (id.RoomID, error)
	CreateRoomAndInviteUser(context.Context, id.UserID) (id.RoomID, error)
	EnableEncryptionForRoom(context.Context, id.RoomID) error
	GetEncryptedRoomForRecipient(context.Context, id.UserID) (id.RoomID, bool)
}
type roomHandler struct {
	client *mautrix.Client
	logger *zap.SugaredLogger
	rooms  map[id.UserID]id.RoomID
	mu     sync.RWMutex
}

func NewRoomHandler(client *mautrix.Client, logger *zap.SugaredLogger) RoomHandler {
	return &roomHandler{client: client, logger: logger, rooms: make(map[id.UserID]id.RoomID)}
}

func (r *roomHandler) GetOrCreateRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, error) {

	// check if room already established with recipient
	roomID, found := r.GetEncryptedRoomForRecipient(ctx, recipient)

	var err error
	// if not create room and invite recipient
	if !found {
		roomID, err = r.CreateRoomAndInviteUser(ctx, recipient)
		if err != nil {
			return "", err
		}
		// enable encryption for room
		err = r.EnableEncryptionForRoom(ctx, roomID)
		if err != nil {
			return "", err
		}
	}

	// return room id
	return roomID, nil
}

func (r *roomHandler) CreateRoomAndInviteUser(ctx context.Context, userID id.UserID) (id.RoomID, error) {
	r.logger.Debugf("Creating room and inviting user %v", userID)
	req := mautrix.ReqCreateRoom{
		Visibility: "private",
		Preset:     "private_chat",
		Invite:     []id.UserID{userID},
	}
	resp, err := r.client.CreateRoom(ctx, &req)
	if err != nil {
		return "", err
	}
	r.cacheRoom(userID, resp.RoomID)
	return resp.RoomID, nil
}

func (r *roomHandler) EnableEncryptionForRoom(ctx context.Context, roomID id.RoomID) error {
	r.logger.Debugf("Enabling encryption for room %s", roomID)
	_, err := r.client.SendStateEvent(ctx, roomID, event.StateEncryption, "",
		event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1})
	return err
}

func (r *roomHandler) GetEncryptedRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, bool) {
	roomID := r.fetchCachedRoom(recipient)
	if roomID != "" {
		return roomID, true
	}
	// if not found query joined rooms
	rooms, err := r.client.JoinedRooms(ctx)
	if err != nil {
		return "", false
	}
	for _, roomID := range rooms.JoinedRooms {
		if encrypted, err := r.client.StateStore.IsEncrypted(ctx, roomID); err != nil || !encrypted {
			continue
		}
		members, err := r.client.JoinedMembers(ctx, roomID)
		if err != nil {
			return "", false
		}

		_, found := members.Joined[recipient]
		if found {
			r.cacheRoom(recipient, roomID)
			return roomID, found
		}
	}
	return "", false
}

func (r *roomHandler) fetchCachedRoom(recipient id.UserID) id.RoomID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rooms[recipient]
}
func (r *roomHandler) cacheRoom(recipient id.UserID, roomID id.RoomID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rooms[recipient] = roomID
}

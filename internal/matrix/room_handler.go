package matrix

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Client interface {
	IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error)
	CreateRoom(ctx context.Context, req *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error)
	SendStateEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string, content interface{}) (*mautrix.RespSendEvent, error)
	JoinedRooms(ctx context.Context) (*mautrix.RespJoinedRooms, error)
	JoinedMembers(ctx context.Context, roomID id.RoomID) (*mautrix.RespJoinedMembers, error)
}

// wrappedClient is a wrapper around mautrix.Client to abstract away concrete implementations and thus facilitate testing and mocking
type wrappedClient struct {
	*mautrix.Client
}

func (c *wrappedClient) IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error) {
	return c.StateStore.IsEncrypted(ctx, roomID)
}

func NewClient(mautrixClient *mautrix.Client) Client {
	return &wrappedClient{mautrixClient}
}

type RoomHandler interface {
	GetOrCreateRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, error)
}
type roomHandler struct {
	client Client
	logger *zap.SugaredLogger
	rooms  map[id.UserID]id.RoomID
	mu     sync.RWMutex
}

func NewRoomHandler(client Client, logger *zap.SugaredLogger) RoomHandler {
	return &roomHandler{client: client, logger: logger, rooms: make(map[id.UserID]id.RoomID)}
}

func (r *roomHandler) GetOrCreateRoomForRecipient(ctx context.Context, recipient id.UserID) (id.RoomID, error) {
	// check if room already established with recipient
	roomID, found := r.getEncryptedRoomForRecipient(ctx, recipient)

	var err error
	// if not create room and invite recipient
	if !found {
		roomID, err = r.createRoomAndInviteUser(ctx, recipient)
		if err != nil {
			return "", err
		}
		// enable encryption for room
		err = r.enableEncryptionForRoom(ctx, roomID)
		if err != nil {
			return "", err
		}
	}

	return roomID, nil
}

func (r *roomHandler) createRoomAndInviteUser(ctx context.Context, userID id.UserID) (id.RoomID, error) {
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

func (r *roomHandler) enableEncryptionForRoom(ctx context.Context, roomID id.RoomID) error {
	r.logger.Debugf("Enabling encryption for room %s", roomID)
	_, err := r.client.SendStateEvent(ctx, roomID, event.StateEncryption, "",
		event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1})
	return err
}

func (r *roomHandler) getEncryptedRoomForRecipient(ctx context.Context, recipient id.UserID) (roomID id.RoomID, found bool) {
	roomID = r.fetchCachedRoom(recipient)
	if roomID != "" {
		return roomID, true
	}
	// if not found query joined rooms
	rooms, err := r.client.JoinedRooms(ctx)
	if err != nil {
		return "", false
	}
	for _, roomID := range rooms.JoinedRooms {
		if encrypted, err := r.client.IsEncrypted(ctx, roomID); err != nil || !encrypted {
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

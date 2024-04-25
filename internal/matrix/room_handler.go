package matrix

import (
	"sync"

	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Client interface {
	IsEncrypted(roomID id.RoomID) bool
	CreateRoom(req *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error)
	SendStateEvent(roomID id.RoomID, eventType event.Type, stateKey string, content interface{}) (*mautrix.RespSendEvent, error)
	JoinedRooms() (*mautrix.RespJoinedRooms, error)
	JoinedMembers(roomID id.RoomID) (*mautrix.RespJoinedMembers, error)
}

// wrappedClient is a wrapper around mautrix.Client to abstract away concrete implementations and thus facilitate testing and mocking
type wrappedClient struct {
	*mautrix.Client
}

func (c *wrappedClient) IsEncrypted(roomID id.RoomID) bool {
	return c.StateStore.IsEncrypted(roomID)
}

func NewClient(mautrixClient *mautrix.Client) Client {
	return &wrappedClient{mautrixClient}
}

type RoomHandler interface {
	GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error)
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

func (r *roomHandler) GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error) {
	// check if room already established with recipient
	roomID, found := r.getEncryptedRoomForRecipient(recipient)

	var err error
	// if not create room and invite recipient
	if !found {
		roomID, err = r.createRoomAndInviteUser(recipient)
		if err != nil {
			return "", err
		}
		// enable encryption for room
		err = r.enableEncryptionForRoom(roomID)
		if err != nil {
			return "", err
		}
	}

	return roomID, nil
}

func (r *roomHandler) createRoomAndInviteUser(userID id.UserID) (id.RoomID, error) {
	r.logger.Debugf("Creating room and inviting user %v", userID)
	req := mautrix.ReqCreateRoom{
		Visibility: "private",
		Preset:     "private_chat",
		Invite:     []id.UserID{userID},
	}
	resp, err := r.client.CreateRoom(&req)
	if err != nil {
		return "", err
	}
	r.cacheRoom(userID, resp.RoomID)
	return resp.RoomID, nil
}

func (r *roomHandler) enableEncryptionForRoom(roomID id.RoomID) error {
	r.logger.Debugf("Enabling encryption for room %s", roomID)
	_, err := r.client.SendStateEvent(roomID, event.StateEncryption, "",
		event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1})
	return err
}

func (r *roomHandler) getEncryptedRoomForRecipient(recipient id.UserID) (roomID id.RoomID, found bool) {
	roomID = r.fetchCachedRoom(recipient)
	if roomID != "" {
		return roomID, true
	}
	// if not found query joined rooms
	rooms, err := r.client.JoinedRooms()
	if err != nil {
		return "", false
	}
	for _, roomID := range rooms.JoinedRooms {
		if !r.client.IsEncrypted(roomID) {
			continue
		}
		members, err := r.client.JoinedMembers(roomID)
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

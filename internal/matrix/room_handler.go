package matrix

import (
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"sync"
)

const RoomPoolSize = 50

type RoomHandler interface {
	Init()
	GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error)
	CacheRoom(recipient id.UserID, roomID id.RoomID)
	HasAlreadyJoined(recipient id.UserID, roomID id.RoomID) bool
}

type roomHandler struct {
	client *mautrix.Client
	logger *zap.SugaredLogger
	rooms  map[id.UserID]*RoomPool
	mu     sync.RWMutex
}

func NewRoomHandler(client *mautrix.Client, logger *zap.SugaredLogger) RoomHandler {
	return &roomHandler{client: client, logger: logger, rooms: make(map[id.UserID]*RoomPool)}
}
func (r *roomHandler) Init() {
	rooms, err := r.client.JoinedRooms()
	if err != nil {
		r.logger.Warn("failed to fetch joined rooms - skipping room handler initialization")
		return
	}

	// pubKeyCache all encrypted rooms
	for _, roomID := range rooms.JoinedRooms {
		if r.client.StateStore.IsEncrypted(roomID) {
			continue
		}
		r.logger.Debugf("Caching room %v | encrypted: %v", roomID, r.client.StateStore.IsEncrypted(roomID))
		members, err := r.client.JoinedMembers(roomID)
		if err != nil {
			r.logger.Debugf("failed to fetch members for room %v", roomID)
			continue
		}
		for userID, _ := range members.Joined {
			r.CacheRoom(userID, roomID)
		}
	}
}

func (r *roomHandler) CacheRoom(recipient id.UserID, roomID id.RoomID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := r.rooms[recipient]; !found {
		r.rooms[recipient] = NewRoomPool(RoomPoolSize)
	}
	pool := r.rooms[recipient]
	pool.Add(roomID)
}

func (r *roomHandler) HasAlreadyJoined(recipient id.UserID, roomID id.RoomID) bool {
	if _, found := r.rooms[recipient]; !found {
		return false
	}
	for _, id := range r.rooms[recipient].rooms {
		if roomID == id {
			return true
		}
	}
	return false
}
func (r *roomHandler) GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error) {

	// check if room already established with recipient
	roomID, _ := r.getEncryptedRoomForRecipient(recipient)

	addNewRoomToPool := func() (id.RoomID, error) {
		roomID, err := r.createRoomAndInviteUser(recipient)
		if err != nil {
			return "", err
		} else {
			//err = r.enableEncryptionForRoom(roomID)
			return roomID, err
		}
	}

	// even if we have found a cached room, we will still create new rooms
	if roomPool, found := r.rooms[recipient]; !found || roomPool.currentSize < RoomPoolSize {
		return addNewRoomToPool()
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
	r.CacheRoom(userID, resp.RoomID)
	return resp.RoomID, nil
}

func (r *roomHandler) enableEncryptionForRoom(roomID id.RoomID) error {
	r.logger.Debugf("Enabling encryption for room %s", roomID)
	_, err := r.client.SendStateEvent(roomID, event.StateEncryption, "",
		event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1, RotationPeriodMessages: 10000})
	return err
}

func (r *roomHandler) getEncryptedRoomForRecipient(recipient id.UserID) (id.RoomID, bool) {
	roomID := r.fetchCachedRoom(recipient)
	if roomID != "" {
		return roomID, true
	}
	// if not found query joined rooms
	rooms, err := r.client.JoinedRooms()
	if err != nil {
		return "", false
	}

	createdRooms := 0
	for _, roomID := range rooms.JoinedRooms {
		if r.client.StateStore.IsEncrypted(roomID) {
			continue
		}
		members, err := r.client.JoinedMembers(roomID)
		if err != nil {
			return "", false
		}

		_, found := members.Joined[recipient]
		if found {
			r.CacheRoom(recipient, roomID)
			createdRooms++
		}
		if createdRooms == RoomPoolSize {
			return r.fetchCachedRoom(recipient), true
		}
	}

	if createdRooms > 0 {
		return r.fetchCachedRoom(recipient), true
	}
	return "", false
}

func (r *roomHandler) fetchCachedRoom(recipient id.UserID) id.RoomID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, found := r.rooms[recipient]; found {
		return r.rooms[recipient].Get()
	}
	return ""
}

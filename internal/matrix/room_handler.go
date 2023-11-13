package matrix

import (
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type RoomHandler interface {
	GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error)
	CreateRoomAndInviteUsers(userIDs []id.UserID) (id.RoomID, error)
	EnableEncryptionForRoom(roomID id.RoomID) error
	GetEncryptedRoomForRecipient(recipient id.UserID) (id.RoomID, bool)
}
type roomHandler struct {
	client *mautrix.Client
	logger *zap.SugaredLogger
}

func NewRoomHandler(client *mautrix.Client, logger *zap.SugaredLogger) RoomHandler {
	return &roomHandler{client: client, logger: logger}
}

func (r *roomHandler) GetOrCreateRoomForRecipient(recipient id.UserID) (id.RoomID, error) {

	// check if room already established with recipient
	roomID, found := r.GetEncryptedRoomForRecipient(recipient)

	var err error
	// if not create room and invite recipient
	if !found {
		roomID, err = r.CreateRoomAndInviteUsers([]id.UserID{recipient})
		if err != nil {
			return "", err
		}
		// enable encryption for room
		err = r.EnableEncryptionForRoom(roomID)
		if err != nil {
			return "", err
		}
	}

	// return room id
	return roomID, nil
}

func (r *roomHandler) CreateRoomAndInviteUsers(userIDs []id.UserID) (id.RoomID, error) {
	r.logger.Debugf("Creating room and inviting users %v", userIDs)
	req := mautrix.ReqCreateRoom{
		Visibility: "private",
		Preset:     "private_chat",
		Invite:     userIDs,
	}
	resp, err := r.client.CreateRoom(&req)
	if err != nil {
		return "", err
	}
	return resp.RoomID, nil
}

func (r *roomHandler) EnableEncryptionForRoom(roomID id.RoomID) error {
	r.logger.Debugf("Enabling encryption for room %s", roomID)
	_, err := r.client.SendStateEvent(roomID, event.StateEncryption, "",
		event.EncryptionEventContent{Algorithm: id.AlgorithmMegolmV1})
	return err
}

func (r *roomHandler) GetEncryptedRoomForRecipient(recipient id.UserID) (id.RoomID, bool) {
	//TODO implement look in cache/memory

	// if not found query joined rooms
	rooms, err := r.client.JoinedRooms()
	if err != nil {
		return "", false
	}
	for _, roomID := range rooms.JoinedRooms {
		if !r.client.StateStore.IsEncrypted(roomID) {
			continue
		}
		members, err := r.client.JoinedMembers(roomID)
		if err != nil {
			return "", false
		}

		_, found := members.Joined[recipient]
		return roomID, found

	}
	return "", false
}

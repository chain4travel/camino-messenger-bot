/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package matrix

import (
	"math/rand"
	"maunium.net/go/mautrix/id"
)

type RoomPool struct {
	rooms       []id.RoomID
	size        int
	currentIdx  int
	currentSize int
}

func NewRoomPool(size int) *RoomPool {
	return &RoomPool{
		rooms:      make([]id.RoomID, size),
		size:       size,
		currentIdx: 0,
	}
}

func (q *RoomPool) Add(roomID id.RoomID) {
	q.rooms[q.currentIdx] = roomID
	q.currentIdx = (q.currentIdx + 1) % q.size
	if q.currentSize < q.size { // else we're just replacing old rooms
		q.currentSize++
	}
}
func (q *RoomPool) Get() id.RoomID {
	if q.currentSize == 0 {
		return ""
	}
	return q.rooms[rand.Intn(q.currentSize)]
}

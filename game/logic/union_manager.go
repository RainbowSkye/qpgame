package logic

import (
	"fmt"
	"game/component/room"
	"math/rand"
	"sync"
	"time"
)

type UnionManager struct {
	sync.RWMutex
	UnionList map[int64]*Union
}

func NewUnionManager() *UnionManager {
	return &UnionManager{
		UnionList: make(map[int64]*Union),
	}
}

func (u *UnionManager) GetUnion(unionId int64) *Union {
	u.Lock()
	defer u.Unlock()
	union, ok := u.UnionList[unionId]
	if ok {
		return union
	}
	union = NewUnion(u)
	u.UnionList[unionId] = union
	return union
}

func (u *UnionManager) CreateRoomId() string {
	roomId := u.genRoomId()
	for _, v := range u.UnionList {
		_, ok := v.RoomList[roomId]
		if ok {
			return u.CreateRoomId()
		}
	}
	return roomId
}

func (u *UnionManager) genRoomId() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randInt := rand.Int63n(899999) + 100000
	return fmt.Sprintf("%d", randInt)
}

func (u *UnionManager) GetRoomById(roomId string) *room.Room {
	for _, union := range u.UnionList {
		if r, ok := union.RoomList[roomId]; ok {
			return r
		}
	}
	return nil
}

package logic

import (
	"core/models/entity"
	"fmt"
	"framework/err"
	"framework/remote"
	"game/component/room"
	"game/models/request"
	"sync"
)

type Union struct {
	sync.RWMutex
	Id       int64
	m        *UnionManager
	RoomList map[string]*room.Room
}

func NewUnion(m *UnionManager) *Union {
	return &Union{
		m:        m,
		RoomList: make(map[string]*room.Room),
	}
}

func (u *Union) CreateRoom(session *remote.Session, req request.CreateRoomReq,
	userData *entity.User) *err.Error {
	// 1.创建一个房间，生成房间号
	roomId := u.m.CreateRoomId()
	fmt.Println("CreateRoom roomID = ", roomId)
	newRoom := room.NewRoom(roomId, req.UnionID, req.GameRule, u)
	u.RoomList[roomId] = newRoom
	return newRoom.UserEntryRoom(session, userData)
}

func (u *Union) DismissRoom(roomId string) {
	u.Lock()
	defer u.Unlock()
	delete(u.RoomList, roomId)
}

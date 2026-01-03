package logic

import (
	"core/models/entity"
	"core/service"
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

func (u *Union) CreateRoom(service *service.UserService, session *remote.Session, req request.CreateRoomReq,
	userData *entity.User) error {
	// 1.创建一个房间，生成房间号
	roomId := u.m.CreateRoomId()
	newRoom := room.NewRoom(roomId, req.UnionID, req.GameRule)
	u.RoomList[roomId] = newRoom
	return newRoom.UserEntryRoom(session, userData)
}

func NewUnion(m *UnionManager) *Union {
	return &Union{
		m:        m,
		RoomList: make(map[string]*room.Room),
	}
}

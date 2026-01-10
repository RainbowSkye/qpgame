package room

import (
	"core/models/entity"
	"framework/remote"
	"game/component/proto"
	"game/component/sz"
	"game/models/request"
	"sync"
)

type Room struct {
	sync.RWMutex
	Id          string
	UnionId     int64
	gameRule    proto.GameRule
	RoomCreator *proto.RoomCreator
	users       map[string]*proto.RoomUser
	GameFrame   GameFrame
}

func (r *Room) UserEntryRoom(session *remote.Session, data *entity.User) error {
	r.RoomCreator = &proto.RoomCreator{
		Uid: data.Uid,
	}
	if r.UnionId == 1 { // 普通玩家创建
		r.RoomCreator.CreatorType = proto.UserCreatorType
	} else { // 联盟创建
		r.RoomCreator.CreatorType = proto.UnionCreatorType
	}
	_, ok := r.users[data.Uid]
	if !ok {
		r.users[data.Uid] = proto.ToRoomUser(data, 1) // todo
	}
	// 2.将房间号推送给客户端 更新数据库 当前房间号存储起来
	r.UpdateUserInfoPush(session, data.Uid)
	session.Put("roomId", r.Id)
	// 3.将游戏类型推送给客户端（用户进入游戏的推送）
	r.SelfEntryRoomPush(session, data.Uid)
	// 4.告诉其他人此用户进入房间了
	return nil
}

func (r *Room) UpdateUserInfoPush(session *remote.Session, uid string) {
	// {
	// 	roomID:'336842',
	// 	pushRouter:'UpdateUserInfoPush'
	// }
	pushMsg := map[string]any{
		"roomID":     r.Id,
		"pushRouter": "UpdateUserInfoPush",
	}
	// node 节点 nats client， push通过nats将消息发送给connector，connector将消息发送给客户端
	// ServerMessagePush
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func (r *Room) SelfEntryRoomPush(session *remote.Session, uid string) {
	// {
	// 	gameType: 1,
	// 	pushRouter:'SelfEntryRoomPush'
	// }
	pushMsg := map[string]any{
		"gameType":   r.gameRule.GameType,
		"pushRouter": "SelfEntryRoomPush",
	}
	// node 节点 nats client， push通过nats将消息发送给connector，connector将消息发送给客户端
	// ServerMessagePush
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func (r *Room) RoomMessageHandle(session *remote.Session, req request.RoomMessageReq) {
	if req.Type == proto.GetRoomSceneInfoNotify {
		r.getRoomSceneInfoPush(session)
	}
}

func (r *Room) getRoomSceneInfoPush(session *remote.Session) {
	roomUserInfoArr := make([]*proto.RoomUser, 0)
	for _, v := range r.users {
		roomUserInfoArr = append(roomUserInfoArr, v)
	}
	data := map[string]any{
		"type":       proto.GetRoomSceneInfoPush,
		"pushRouter": "RoomMessagePush",
		"data": map[string]any{
			"roomId":          r.Id,
			"roomCreatorInfo": r.RoomCreator,
			"gameRule":        r.gameRule,
			"roomUserInfoArr": roomUserInfoArr,
			"gameData":        r.GameFrame.GetGameData(session),
		},
	}
	session.Push([]string{session.GetUid()}, data, "ServerMessagePush")
}

func NewRoom(roomId string, unionId int64, gameRule proto.GameRule) *Room {
	r := &Room{
		Id:       roomId,
		UnionId:  unionId,
		gameRule: gameRule,
		users:    make(map[string]*proto.RoomUser),
	}
	if gameRule.GameType == int(proto.PinSanZhang) {
		r.GameFrame = sz.NewGameFrame(gameRule, r)
	}
	return r
}

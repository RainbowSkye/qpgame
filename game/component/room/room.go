package room

import (
	"core/models/entity"
	"framework/remote"
	"game/component/proto"
	"sync"
)

type Room struct {
	sync.RWMutex
	Id       string
	UnionId  int64
	gameRule proto.GameRule
}

func (r *Room) UserEntryRoom(session *remote.Session, data *entity.User) error {
	// 2.将房间号推送给客户端 更新数据库 当前房间号存储起来
	r.UpdateUserInfoPush(session, data.Uid)
	// 3.将游戏类型推送给客户端（用户进入游戏的推送）

	// 4.告诉其他人此用户进入房间了
	return nil
}

func (r *Room) UpdateUserInfoPush(session *remote.Session, uid string) {
	// {
	// 	roomID:'336842',
	// 	pushRouter:'updateUserInfoPush'
	// }
	pushMsg := map[string]any{
		"roomID":     r.Id,
		"pushRouter": "UpdateUserInfoPush",
	}
	// node 节点 nats client， push通过nats将消息发送给connector，connector将消息发送给客户端
	// ServerMessagePush
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func NewRoom(roomId string, unionId int64, gameRule proto.GameRule) *Room {
	return &Room{
		Id:       roomId,
		UnionId:  unionId,
		gameRule: gameRule,
	}
}

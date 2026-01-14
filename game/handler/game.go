package handler

import (
	"common"
	"common/biz"
	"core/repo"
	"core/service"
	"encoding/json"
	"fmt"
	"framework/remote"
	"game/logic"
	"game/models/request"
)

type GameHandler struct {
	um          *logic.UnionManager
	userService *service.UserService
}

func NewGameHandler(r *repo.Manager, um *logic.UnionManager) *GameHandler {
	return &GameHandler{
		um:          um,
		userService: service.NewUserService(r),
	}
}

func (g *GameHandler) RoomMessageNotify(session *remote.Session, msg []byte) any {
	if len(session.GetUid()) <= 0 {
		common.F(biz.InvalidUsers)
	}

	var req request.RoomMessageReq
	err := json.Unmarshal(msg, &req)
	if err != nil {
		common.F(biz.RequestDataError)
	}

	roomId, ok := session.Get("roomId")
	if !ok {
		common.F(biz.NotInRoom)
	}
	room := g.um.GetRoomById(fmt.Sprintf("%v", roomId))
	if room == nil {
		return common.F(biz.NotInRoom)
	}
	room.RoomMessageHandle(session, req)

	return nil
}

func (g *GameHandler) GameMessageNotify(session *remote.Session, msg []byte) any {
	if len(session.GetUid()) <= 0 {
		common.F(biz.InvalidUsers)
	}

	roomId, ok := session.Get("roomId")
	if !ok {
		common.F(biz.NotInRoom)
	}
	room := g.um.GetRoomById(fmt.Sprintf("%v", roomId))
	if room == nil {
		return common.F(biz.NotInRoom)
	}
	room.GetMessageHandle(session, msg)
	return nil
}

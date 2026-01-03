package handler

import (
	"common"
	"common/biz"
	"context"
	"core/repo"
	"core/service"
	"encoding/json"
	"framework/remote"
	"game/logic"
	"game/models/request"
)

type UnionHandler struct {
	um          *logic.UnionManager
	userService *service.UserService
}

func (u *UnionHandler) CreateRoom(session *remote.Session, msg []byte) any {
	// union 联盟 - 持有房间
	// unionManager 管理联盟
	// room 房间 - 关联 game 接口 实现多个不同的游戏

	// 1.接受参数
	uid := session.GetUid()
	if len(uid) <= 0 {
		return common.F(biz.InvalidUsers)
	}

	var req request.CreateRoomReq
	err := json.Unmarshal(msg, &req)
	if err != nil {
		return common.F(biz.RequestDataError)
	}

	// 2.根据session 用户id 查询用户信息
	userData, err := u.userService.FindUserByUid(context.Background(), uid)
	if err != nil {
		return common.F(biz.SqlError)
	}
	if userData == nil {
		return common.F(biz.InvalidUsers)
	}

	// 3.根据游戏规则、游戏类型、用户信息（创建房间的用户）创建房间
	// TODO 需要判断 session 中是否已经有roomI， 如果有代表此用户已经在房价中了， 就不能再次创建房间了
	union := u.um.GetUnion(req.UnionID)
	err = union.CreateRoom(u.userService, session, req, userData)
	if err != nil {
		return common.F(biz.UnionNotExist)
	}

	return common.S(nil)
}

func NewUnionHandler(r *repo.Manager, um *logic.UnionManager) *UnionHandler {
	return &UnionHandler{
		um:          um,
		userService: service.NewUserService(r),
	}
}

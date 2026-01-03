package handler

import (
	"common"
	"common/biz"
	"common/config"
	"common/jwts"
	"connector/models/request"
	"context"
	"core/repo"
	"core/service"
	"encoding/json"
	"fmt"
	"framework/game"
	"framework/net"

	"go.uber.org/zap"
)

type EntryHandler struct {
	userService *service.UserService
}

func (h *EntryHandler) Entry(session *net.Session, body []byte) (any, error) {
	zap.L().Info("==============Entry Start=====================")
	zap.L().Info("entry request params: " + string(body))
	zap.L().Info("==============Entry End=====================")
	var req request.EntryReq
	err := json.Unmarshal(body, &req)
	if err != nil {
		return common.F(biz.RequestDataError), nil
	}
	// 校验token
	uid, err := jwts.ParseToken(req.Token, config.Conf.Jwt.Secret)
	if err != nil {
		zap.L().Error("parse token err: ", zap.Error(err))
		return common.F(biz.TokenInfoError), nil
	}
	// 根据uid 去mongo中查询用户 如果用户不存在 生成一个用户
	user, err := h.userService.FindAndSaveUserByUid(context.TODO(), uid, req.UserInfo)
	if err != nil {
		return common.F(biz.SqlError), nil
	}
	fmt.Printf("session = %v\n", session)
	session.Uid = uid
	return common.S(map[string]any{
		"userInfo": user,
		"config":   game.Conf.GetFrontGameConfig(),
	}), nil
}

func NewEntryHandler(r *repo.Manager) *EntryHandler {
	return &EntryHandler{
		userService: service.NewUserService(r),
	}
}

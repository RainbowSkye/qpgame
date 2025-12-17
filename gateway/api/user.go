package api

import (
	"common"
	"common/biz"
	"common/config"
	"common/jwts"
	"common/rpc"
	"errors"
	err2 "framework/err"
	"time"
	"user/pb"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type UserHandler struct {
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (u *UserHandler) Register(ctx *gin.Context) {
	var req pb.RegisterParams
	err := ctx.Bind(&req)
	if err != nil {
		zap.L().Error("Register 绑定参数失败, err: ", zap.Error(err))
		common.Fail(ctx, biz.RequestDataError)
		return
	}

	resp, err := rpc.UserGrpcClient.Register(ctx, &req)
	if err != nil {
		zap.L().Error("注册失败, err: ", zap.Error(err))
		common.Fail(ctx, err2.ToError(err))
		return
	}
	// 生成 jwt token
	token, err := jwts.GenToken(&jwts.CustomClaims{
		Uid: resp.Uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}, config.Conf.Jwt.Secret)
	if err != nil {
		zap.L().Error("生成 jwt token 失败, err: ", zap.Error(err))
		common.Fail(ctx, err2.NewError(1, errors.New("生成 jwt token 失败")))
		return
	}

	result := map[string]any{
		"token": token,
		"serverInfo": map[string]any{
			"host": config.Conf.Services["connector"].ClientHost,
			"port": config.Conf.Services["connector"].ClientPort,
		},
	}

	common.Success(ctx, result)
}

package common

import (
	"common/biz"
	"framework/err"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Result struct {
	Code int `json:"code"`
	Msg  any `json:"msg"`
}

func Success(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, Result{
		Code: biz.OK,
		Msg:  data,
	})
}

func Fail(ctx *gin.Context, err *err.Error) {
	ctx.JSON(http.StatusOK, Result{
		Code: err.Code,
		Msg:  err.Error(),
	})
}

func F(err *err.Error) Result {
	return Result{
		Code: err.Code,
	}
}

func S(data any) Result {
	return Result{
		Code: biz.OK,
		Msg:  data,
	}
}

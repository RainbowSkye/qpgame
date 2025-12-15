package router

import (
	"common/config"
	"gateway/api"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	if config.Conf.Log.Level == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	userHandler := api.NewUserHandler()
	r.POST("/register", userHandler.Register)

	return r
}

package router

import (
	"common/config"
	"gateway/api"
	"gateway/auth"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	if config.Conf.Log.Level == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(auth.Cors())
	userHandler := api.NewUserHandler()
	r.POST("/register", userHandler.Register)

	return r
}

package app

import (
	"common/config"
	"common/logs"
	"context"
	"fmt"
	"gateway/router"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Run 启动各种服务
func Run(ctx context.Context) error {
	// 初始化日志服务
	logs.InitLogger(&config.Conf.Log)
	zap.L().Info("初始化日志...")

	go func() {
		// 启动gin路由
		r := router.InitRouter()
		if err := r.Run(fmt.Sprintf(":%d", config.Conf.HttpPort)); err != nil {
			zap.L().Error("启动gin失败，err: ", zap.Error(err))
			panic(err)
		}
	}()

	stop := func() {
		zap.L().Info("stop server")
		time.Sleep(1 * time.Second)
	}

	// 优雅启停
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			stop()
			zap.L().Info("ctx done")
			return nil
		case sig := <-ch:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				stop()
				zap.L().Info("stop by " + sig.String())
				return nil
			case syscall.SIGHUP:
				stop()
				zap.L().Info("stop by " + sig.String())
				return nil
			default:
				return nil
			}

		}
	}
}

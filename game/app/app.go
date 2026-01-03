package app

import (
	"common/config"
	"common/logs"
	"context"
	"core/repo"
	"framework/node"
	"game/route"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func Run(ctx context.Context, serverId string) error {
	// 初始化日志服务
	logs.InitLogger(&config.Conf.Log)
	zap.L().Info("初始化日志...")
	exit := func() {}
	go func() {
		n := node.Default()
		exit = n.Close
		manager := repo.New()
		n.RegisterHandler(route.Register(manager))
		n.Run(serverId)
	}()
	stop := func() {
		// other
		exit()
		time.Sleep(3 * time.Second)
		zap.L().Info("stop app finish")
	}
	// 期望有一个优雅启停 遇到中断 退出 终止 挂断
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			stop()
			// time out
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				stop()
				zap.L().Info("connector app quit")
				return nil
			case syscall.SIGHUP:
				stop()
				zap.L().Info("hang up!! connector app quit")
				return nil
			default:
				return nil
			}
		}
	}
}

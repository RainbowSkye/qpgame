package app

import (
	"common/config"
	"common/logs"
	"connector/route"
	"context"
	"core/repo"
	"framework/connector"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Run 启动各种服务
func Run(ctx context.Context, serverId string) error {
	// 初始化日志服务
	logs.InitLogger(&config.Conf.Log)
	zap.L().Info("初始化日志...")

	exit := func() {}
	go func() {
		c := connector.Default()
		exit = c.Close
		manager := repo.New()
		c.RegisterHandler(route.Register(manager))
		c.Run(serverId)
	}()

	stop := func() {
		exit()
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

package app

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"core/repo"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Run 启动各种服务
func Run(ctx context.Context) error {
	// 初始化日志服务
	logs.InitLogger(&config.Conf.Log)
	zap.L().Info("初始化日志...")

	// 启动grpc服务
	server := grpc.NewServer()

	// 初始化数据库连接 mongo、redis
	manager := repo.New()

	go func() {
		lis, err := net.Listen("tcp", config.Conf.Grpc.Addr)
		if err != nil {
			zap.L().Error("listen tcp fail, err: ", zap.Error(err))
			panic(err)
		}

		// 注册到etcd
		register := discovery.NewRegister(config.Conf.Etcd.Addrs)
		_, err = register.Register(&config.Conf.Etcd.Register)
		if err != nil {
			zap.L().Error("register etcd fail, err: ", zap.Error(err))
			panic(err)
		}

		if err = server.Serve(lis); err != nil {
			zap.L().Error("grpc server fail, err: ", zap.Error(err))
			panic(err)
		}
	}()

	stop := func() {
		zap.L().Info("stop server")
		manager.Close()
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

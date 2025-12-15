package database

import (
	"common/config"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisManager struct {
	Client redis.Cmdable
}

func NewRedis() *RedisManager {
	var cli redis.Cmdable
	clusterAddrs := config.Conf.Database.RedisConf.ClusterAddrs
	if len(clusterAddrs) == 0 { // 单节点
		cli = redis.NewClient(&redis.Options{
			Addr:         config.Conf.Database.RedisConf.Addr,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	} else { // 集群
		cli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Conf.Database.RedisConf.ClusterAddrs,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	}

	if err := cli.Ping(context.Background()).Err(); err != nil {
		zap.L().Error("ping redis failed, err: ", zap.Error(err))
		panic(err)
	}

	return &RedisManager{
		Client: cli,
	}
}

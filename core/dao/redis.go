package dao

import (
	"context"
	"core/repo"

	"go.uber.org/zap"
)

const (
	Prefix            = "chess"
	AccountIdRedisKey = "AccountId"
	AccountIdBegin    = 10000
)

type RedisDao struct {
	repo *repo.Manager
}

func NewRedisDao(repo *repo.Manager) *RedisDao {
	return &RedisDao{
		repo: repo,
	}
}

func (r *RedisDao) NextAccount(ctx context.Context) (int64, error) {
	return r.incr(ctx, Prefix+":"+AccountIdRedisKey)
}

func (r *RedisDao) incr(ctx context.Context, key string) (int64, error) {
	// 先判断一下key是否存在
	exist, err := r.repo.Redis.Client.Exists(ctx, key).Result()
	if exist == 0 { // key 不存在
		// 设置 key 永不过期
		err = r.repo.Redis.Client.Set(ctx, key, AccountIdBegin, 0).Err()
		if err != nil {
			zap.L().Error("设置redis key失败, err: ", zap.Error(err))
			return 0, err
		}
	}
	res, err := r.repo.Redis.Client.Incr(ctx, key).Result()
	if err != nil {
		zap.L().Error("redis key 自增失败, err: ", zap.Error(err))
		return 0, err
	}
	return res, nil
}

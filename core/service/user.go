package service

import (
	"common/utils"
	"connector/models/request"
	"context"
	"core/dao"
	"core/model/entity"
	"core/repo"
	"fmt"
	"framework/game"
	"time"

	"go.uber.org/zap"
)

type UserService struct {
	userDao *dao.UserDao
}

func (s *UserService) FindUserByUid(ctx context.Context, uid string, info request.UserInfo) (*entity.User, error) {
	// 查询mongo 有 返回 没有 新增
	user, err := s.userDao.FindUserByUid(ctx, uid)
	if err != nil {
		zap.L().Error("[UserService] FindUserByUid  user err: ", zap.Error(err))
		return nil, err
	}
	if user == nil {
		// save
		user = &entity.User{}
		user.Uid = uid
		zap.L().Info("game.Conf.GameConfig = ", zap.Any("config", game.Conf.GameConfig))
		user.Gold = int64(game.Conf.GameConfig["startgold"]["value"].(float64))
		user.Avatar = utils.Default(info.Avatar, "Common/head_icon_default")
		user.Nickname = utils.Default(info.Nickname, fmt.Sprintf("%s%s", "码神", uid))
		user.Sex = info.Sex // 0 男 1 女
		user.CreateTime = time.Now().UnixMilli()
		user.LastLoginTime = time.Now().UnixMilli()
		err = s.userDao.Insert(context.TODO(), user)
		if err != nil {
			zap.L().Error("[UserService] FindUserByUid insert user err: ", zap.Error(err))
			return nil, err
		}
	}
	return user, nil
}

func NewUserService(r *repo.Manager) *UserService {
	return &UserService{
		userDao: dao.NewUserDao(r),
	}
}

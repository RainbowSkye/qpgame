package service

import (
	"common/biz"
	"context"
	"core/dao"
	"core/models/entity"
	"core/models/requests"
	"core/repo"
	"fmt"
	err2 "framework/err"
	"time"
	"user/pb"

	"go.uber.org/zap"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	accountDao *dao.AccountDao
	redisDao   *dao.RedisDao
}

func NewUserService(m *repo.Manager) *UserService {
	return &UserService{
		accountDao: dao.NewAccount(m),
		redisDao:   dao.NewRedisDao(m),
	}
}

func (u *UserService) Register(ctx context.Context, req *pb.RegisterParams) (*pb.RegisterResponse, error) {
	var account *entity.Account
	var err error
	if req.LoginPlatform == requests.WeiXin { // 微信注册
		account, err = u.wxRegister(ctx, req)
		if err != nil {
			zap.L().Error("wxRegister插入数据库失败, err: ", zap.Error(err))
			return &pb.RegisterResponse{}, err2.GrpcError(biz.SqlError)
		}
	}

	return &pb.RegisterResponse{
		Uid: account.Uid,
	}, nil
}

// 微信注册
func (u *UserService) wxRegister(ctx context.Context, req *pb.RegisterParams) (*entity.Account, error) {
	// 从 redis 获取 uid
	uid, err := u.redisDao.NextAccount(ctx)
	if err != nil {
		return &entity.Account{}, biz.RedisError
	}
	account := &entity.Account{
		Uid:        fmt.Sprintf("%d", uid),
		WxAccount:  req.Account,
		CreateTime: time.Now(),
	}
	err = u.accountDao.SaveAccount(ctx, account)
	if err != nil {
		return &entity.Account{}, err
	}
	return account, nil
}

func (u *UserService) FindUserByUid(ctx context.Context, req *pb.UserParams) (*pb.UserDTO, error) {
	return &pb.UserDTO{}, nil
}

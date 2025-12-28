package dao

import (
	"context"
	"core/model/entity"
	"core/repo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type UserDao struct {
	repo *repo.Manager
}

func (u *UserDao) FindUserByUid(ctx context.Context, uid string) (*entity.User, error) {
	db := u.repo.Mongo.Db.Collection("user")
	singleResult := db.FindOne(ctx, bson.M{
		"uid": uid,
	})
	var user entity.User
	err := singleResult.Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		zap.L().Error("ENTER FindUserByUid", zap.String("uid", uid))
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) Insert(ctx context.Context, user *entity.User) error {
	db := u.repo.Mongo.Db.Collection("user")
	_, err := db.InsertOne(ctx, user)
	return err
}

func NewUserDao(r *repo.Manager) *UserDao {
	return &UserDao{
		repo: r,
	}
}

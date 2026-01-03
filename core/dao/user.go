package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
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
			zap.L().Info("uid 不存在")
			return nil, nil
		}
		zap.L().Error("ENTER FindUserByUid", zap.String("uid", uid))
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) Insert(ctx context.Context, user *entity.User) error {
	db := u.repo.Mongo.Db.Collection("user")
	log.Printf("user = %+v", user)
	_, err := db.InsertOne(ctx, user)
	zap.L().Info("INSERT user", zap.Any("user", user))
	return err
}

func (u *UserDao) UpdateUserAddress(ctx context.Context, user *entity.User) error {
	db := u.repo.Mongo.Db.Collection("user")
	_, err := db.UpdateOne(ctx, bson.M{
		"uid": user.Uid,
	}, bson.M{
		"$set": bson.M{
			"address":  user.Address,
			"location": user.Location,
		},
	})
	return err
}

func NewUserDao(r *repo.Manager) *UserDao {
	return &UserDao{
		repo: r,
	}
}

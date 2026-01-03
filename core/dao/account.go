package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
)

type AccountDao struct {
	repo *repo.Manager
}

func NewAccount(r *repo.Manager) *AccountDao {
	return &AccountDao{
		repo: r,
	}
}

func (a *AccountDao) SaveAccount(ctx context.Context, ac *entity.Account) error {
	table := a.repo.Mongo.Db.Collection("account")
	_, err := table.InsertOne(ctx, ac)
	return err
}

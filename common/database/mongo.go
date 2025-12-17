package database

import (
	"common/config"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.uber.org/zap"
)

type MongoManager struct {
	Cli *mongo.Client
	Db  *mongo.Database
}

func NewMongo() *MongoManager {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(config.Conf.Database.MongoConf.Url)
	opts.SetAuth(options.Credential{
		Username:   config.Conf.Database.MongoConf.UserName,
		Password:   config.Conf.Database.MongoConf.Password,
		AuthSource: "admin",
	})
	opts.SetMinPoolSize(uint64(config.Conf.Database.MongoConf.MinPoolSize))
	opts.SetMaxPoolSize(uint64(config.Conf.Database.MongoConf.MaxPoolSize))
	client, err := mongo.Connect(opts)
	if err != nil {
		zap.L().Error("connect mongo failed, err: ", zap.Error(err))
		panic(err)
	}

	if err = client.Ping(ctx, &readpref.ReadPref{}); err != nil {
		zap.L().Error("ping mongo failed, err: ", zap.Error(err))
		panic(err)
	}

	return &MongoManager{
		Cli: client,
		Db:  client.Database(config.Conf.Database.MongoConf.Db),
	}
}

func (m *MongoManager) Close() {
	if m.Cli != nil {
		m.Cli.Disconnect(context.Background())
	}
}

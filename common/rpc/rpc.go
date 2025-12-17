package rpc

import (
	"common/config"
	"common/discovery"
	"fmt"
	"user/pb"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var (
	UserGrpcClient pb.UserServiceClient
)

func Init() {
	fmt.Println("初始化grpc客户端...")
	userDomain := config.Conf.Domain["user"]
	initClient(&userDomain, &UserGrpcClient)
}

func initClient(domain *config.Domain, client interface{}) {
	fmt.Println("初始化 user grpc 客户端...")
	etcdRegister := discovery.NewResolver(config.Conf.Etcd.Addrs, zap.L())
	resolver.Register(etcdRegister)

	addr := fmt.Sprintf("etcd:///%s", domain.Name)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if domain.LoadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		zap.L().Error("连接etcd客户端失败, err: ", zap.Error(err))
		panic(err)
	}

	switch c := client.(type) {
	case *pb.UserServiceClient:
		fmt.Println("初始化grpc客户端成功")
		*c = pb.NewUserServiceClient(conn)
	default:
		fmt.Println("没有匹配的类型")
	}
}

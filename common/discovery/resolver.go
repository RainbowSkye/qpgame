package discovery

import (
	"context"
	"log"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
)

const (
	scheme = "etcd"
)

type Resolver struct {
	scheme      string
	EtcdAddrs   []string
	DialTimeout int

	closeCh     chan struct{}
	watchCh     clientv3.WatchChan
	cli         *clientv3.Client
	keyPrefix   string
	svcAddrList []resolver.Address

	cc     resolver.ClientConn
	logger *zap.Logger
}

func NewResolver(etcdAddrs []string, logger *zap.Logger) *Resolver {
	return &Resolver{
		scheme:      scheme,
		EtcdAddrs:   etcdAddrs,
		DialTimeout: 3,
		logger:      logger,
	}
}

func (r *Resolver) Scheme() string {
	return r.scheme
}

func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn,
	opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.cc = cc
	r.keyPrefix = BuildPrefix(Server{Name: target.Endpoint(), Version: target.URL.Host})
	r.logger.Info("target.Endpoint() = " + target.Endpoint() + ", target.URL.Host = " + target.URL.Host)
	r.logger.Info("正在尝试从 etcd 解析服务, keyPrefix = " + r.keyPrefix)
	if _, err := r.start(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Resolver) ResolveNow(o resolver.ResolveNowOptions) {

}

func (r *Resolver) Close() {
	r.closeCh <- struct{}{}
}

func (r *Resolver) start() (chan<- struct{}, error) {
	var err error
	r.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.EtcdAddrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	resolver.Register(r)

	r.closeCh = make(chan struct{})

	if err = r.sync(); err != nil {
		return nil, err
	}

	go r.watch()

	return r.closeCh, nil
}

// watch update events
func (r *Resolver) watch() {
	ticker := time.NewTicker(time.Minute)
	r.watchCh = r.cli.Watch(context.Background(), r.keyPrefix, clientv3.WithPrefix())

	for {
		select {
		case <-r.closeCh:
			return
		case res, ok := <-r.watchCh:
			if ok {
				r.update(res.Events)
			}
		case <-ticker.C:
			if err := r.sync(); err != nil {
				r.logger.Error("sync failed", zap.Error(err))
			}
		}
	}
}

// update
func (r *Resolver) update(events []*clientv3.Event) {
	for _, ev := range events {
		var info Server
		var err error

		switch ev.Type {
		case mvccpb.PUT:
			info, err = ParseValue(ev.Kv.Value)
			if err != nil {
				continue
			}
			addr := resolver.Address{Addr: info.Addr, Metadata: info.Weight}
			if !Exist(r.svcAddrList, addr) {
				r.svcAddrList = append(r.svcAddrList, addr)
				r.cc.UpdateState(resolver.State{Addresses: r.svcAddrList})
			}
		case mvccpb.DELETE:
			info, err = SplitPath(string(ev.Kv.Key))
			if err != nil {
				continue
			}
			addr := resolver.Address{Addr: info.Addr}
			if s, ok := Remove(r.svcAddrList, addr); ok {
				r.svcAddrList = s
				r.cc.UpdateState(resolver.State{Addresses: r.svcAddrList})
			}
		}
	}
}

// sync 同步获取所有地址信息
func (r *Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := r.cli.Get(ctx, r.keyPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Println(err.Error())
		return err
	}

	r.svcAddrList = []resolver.Address{}

	for _, v := range res.Kvs {
		info, err := ParseValue(v.Value)
		if err != nil {
			continue
		}
		addr := resolver.Address{Addr: info.Addr, Metadata: info.Weight}
		r.svcAddrList = append(r.svcAddrList, addr)
	}
	err = r.cc.UpdateState(resolver.State{Addresses: r.svcAddrList})
	if err != nil {
		log.Println("163 " + err.Error())
		return err
	}
	return nil
}

package discovery

import (
	"common/config"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Register struct {
	EtcdAddrs   []string
	DialTimeout int

	closeCh  chan struct{}
	leasesId clientv3.LeaseID

	svcInfo Server
	cli     *clientv3.Client
}

func NewRegister(etcdAddrs []string) *Register {
	return &Register{
		EtcdAddrs:   etcdAddrs,
		DialTimeout: 3,
	}
}

// Register 注册节点
func (r *Register) Register(conf *config.RegisterServer) (chan<- struct{}, error) {
	svcInfo := Server{
		Name:    conf.Name,
		Addr:    conf.Addr,
		Weight:  conf.Weight,
		Version: conf.Version,
		Ttl:     conf.Ttl,
	}
	var err error

	if strings.Split(svcInfo.Addr, ":")[0] == "" {
		return nil, errors.New("invalid ip")
	}

	r.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.EtcdAddrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	r.svcInfo = svcInfo
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()
	leases, err := r.cli.Grant(ctx, svcInfo.Ttl)
	if err != nil {
		return nil, err
	}
	r.leasesId = leases.ID
	data, err := json.Marshal(svcInfo)
	if err != nil {
		return nil, err
	}

	_, err = r.cli.Put(context.Background(), BuildPrefix(r.svcInfo), string(data), clientv3.WithLease(leases.ID))
	if err != nil {
		return nil, err
	}
	log.Println(BuildPrefix(r.svcInfo), string(data))
	r.closeCh = make(chan struct{})

	go r.keepAlive()

	return r.closeCh, nil
}

func (r *Register) Close() {
	r.closeCh <- struct{}{}
}

// 删除节点
func (r *Register) unregister() error {
	_, err := r.cli.Delete(context.Background(), BuildPrefix(r.svcInfo))
	return err
}

func (r *Register) keepAlive() {
	ch, err := r.cli.KeepAlive(context.Background(), r.leasesId)
	if err != nil {
		zap.L().Error("keepAlive failed", zap.Error(err))
	}

	for {
		select {
		case <-r.closeCh:
			if err := r.unregister(); err != nil {
				zap.L().Error("unregister failed", zap.Error(err))
			}
			// 撤销 leasesID
			if _, err := r.cli.Revoke(context.Background(), r.leasesId); err != nil {
				zap.L().Error("revoke leasesId failed", zap.Error(err))
			}
			return
		case _, ok := <-ch:
			if !ok {
				zap.L().Warn("keepalive channel closed")
				// 可以选择重连或退出，根据需求
				return
			}
		}
	}
}

func (r *Register) UpdateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		wi := req.URL.Query().Get("weight")
		weight, err := strconv.Atoi(wi)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		r.svcInfo.Weight = weight
		data, err := json.Marshal(r.svcInfo)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		_, err = r.cli.Put(context.Background(), BuildPrefix(r.svcInfo), string(data), clientv3.WithLease(r.leasesId))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("update server weight success"))
	}
}

func (r *Register) GetServerInfo() (Server, error) {
	resp, err := r.cli.Get(context.Background(), BuildPrefix(r.svcInfo))
	if err != nil {
		return r.svcInfo, err
	}
	var info Server
	if resp.Count > 1 {
		if err = json.Unmarshal(resp.Kvs[0].Value, &info); err != nil {
			return r.svcInfo, err
		}
	}
	return info, nil
}

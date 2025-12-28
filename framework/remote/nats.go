package remote

import (
	"framework/game"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type NatsClient struct {
	serverId string
	conn     *nats.Conn
	readChan chan []byte
}

func NewNatsClient(serverId string, readChan chan []byte) *NatsClient {
	return &NatsClient{
		serverId: serverId,
		readChan: readChan,
	}
}

func (n *NatsClient) Run() error {
	var err error
	n.conn, err = nats.Connect(game.Conf.ServersConf.Nats.Url)
	if err != nil {
		zap.L().Error("connect nats server fail, err: ", zap.Error(err))
		return err
	}
	go n.sub()
	return nil
}

// 订阅
func (n *NatsClient) sub() {
	_, err := n.conn.Subscribe(n.serverId, func(msg *nats.Msg) {
		// 收到的其他nats client发送的消息
		n.readChan <- msg.Data
	})
	if err != nil {
		zap.L().Error("nats sub err: ", zap.Error(err))
	}
}

func (n *NatsClient) SendMsg(dst string, data []byte) error {
	if n.conn != nil {
		return n.conn.Publish(dst, data)
	}
	return nil
}

func (n *NatsClient) Close() error {
	if n.conn != nil {
		n.conn.Close()
	}
	return nil
}

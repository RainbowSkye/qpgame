package connector

import (
	"fmt"
	"framework/game"
	"framework/net"
	"framework/remote"

	"go.uber.org/zap"
)

type Connector struct {
	isRunning bool
	wsManager *net.Manager
	handlers  net.LogicHandler
	remoteCli remote.Client
}

func Default() *Connector {
	return &Connector{}
}

func (c *Connector) Run(serverId string) {
	if !c.isRunning {
		// 启动websocket和nats
		c.wsManager = net.NewManager()
		c.wsManager.ConnectorHandlers = c.handlers
		// 启动nats nats server不会存储消息
		c.remoteCli = remote.NewNatsClient(serverId, c.wsManager.RemoteReadChan)
		c.remoteCli.Run()
		c.wsManager.RemoteCli = c.remoteCli
		c.Serve(serverId)
	}
}

func (c *Connector) Serve(serverId string) {
	connectorConfig := game.Conf.GetConnector(serverId)
	if connectorConfig == nil {
		zap.L().Fatal("connectorConfig is nil")
	}
	c.isRunning = true
	c.wsManager.ServerId = serverId
	addr := fmt.Sprintf("%s:%d", connectorConfig.Host, connectorConfig.ClientPort)
	c.wsManager.Run(addr)
}

func (c *Connector) Close() {
	if c.wsManager != nil {
		c.wsManager.Close()
	}
}

func (c *Connector) RegisterHandler(handlers net.LogicHandler) {
	c.handlers = handlers
}

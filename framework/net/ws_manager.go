package net

import (
	"common/utils"
	"encoding/json"
	"errors"
	"fmt"
	"framework/game"
	"framework/protocol"
	"framework/remote"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	websocketUpgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type CheckOriginHandler func(r *http.Request) bool

type HandlerFunc func(session *Session, body []byte) (any, error)

type LogicHandler map[string]HandlerFunc

type EventHandler func(packet *protocol.Packet, c Connection) error

type Manager struct {
	sync.RWMutex
	websocketUpgrade   *websocket.Upgrader
	ServerId           string // 在connector赋值
	CheckOriginHandler CheckOriginHandler
	clients            map[string]Connection
	RemoteCli          remote.Client // 在connector赋值
	handlers           map[protocol.PackageType]EventHandler
	ConnectorHandlers  LogicHandler // 在connector赋值
	ClientReadChan     chan *MsgPack
	RemoteReadChan     chan []byte
	RemotePushChan     chan *remote.Msg
}

func NewManager() *Manager {
	return &Manager{
		ClientReadChan: make(chan *MsgPack, 1024),
		clients:        make(map[string]Connection),
		handlers:       make(map[protocol.PackageType]EventHandler),
		RemoteReadChan: make(chan []byte, 1024),
		RemotePushChan: make(chan *remote.Msg, 1024),
	}
}

func (m *Manager) Run(addr string) {
	// 设置不同的消息处理器
	m.setupEventHandlers()

	go m.clientReadChanHandler()
	go m.remoteReadChanHandler()
	go m.remotePushChanHandler()
	http.HandleFunc("/", m.serveWs)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		zap.L().Fatal("connector listen serve err: ", zap.Error(err))
	}
}

func (m *Manager) serveWs(w http.ResponseWriter, r *http.Request) {
	if m.websocketUpgrade == nil {
		m.websocketUpgrade = &websocketUpgrade
	}
	// http 服务升级为 websocket
	wsConn, err := m.websocketUpgrade.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Fatal("websocket upgrade failed, err: ", zap.Error(err))
	}
	client := NewWsConnection(wsConn, m)
	m.addClient(client)
	client.Run()
}

func (m *Manager) addClient(client *WsConnection) {
	m.Lock()
	defer m.Unlock()
	m.clients[client.Cid] = client
}

func (m *Manager) removeClient(wc *WsConnection) {
	m.Lock()
	defer m.Unlock()
	wc.Close()
	delete(m.clients, wc.Cid)
}

func (m *Manager) Close() {
	for cid, v := range m.clients {
		v.Close()
		delete(m.clients, cid)
	}
}

// 解析并读取前端传来的数据
func (m *Manager) clientReadChanHandler() {
	for {
		select {
		case data, ok := <-m.ClientReadChan:
			if ok { // 解析数据
				m.decodeClientPack(data)
			}
		}
	}
}

// 解析协议
func (m *Manager) decodeClientPack(data *MsgPack) {
	// zap.L().Info("receiver message " + string(data.Body))
	packet, err := protocol.Decode(data.Body)
	if err != nil {
		zap.L().Error("decode message err: ", zap.Error(err))
		return
	}
	if err = m.routeEvent(packet, data.Cid); err != nil {
		zap.L().Error("routeEvent err: ", zap.Error(err))
	}
}

func (m *Manager) setupEventHandlers() {
	m.handlers[protocol.Handshake] = m.HandshakeHandler
	m.handlers[protocol.HandshakeAck] = m.HandshakeAckHandler
	m.handlers[protocol.Heartbeat] = m.HeartbeatHandler
	m.handlers[protocol.Data] = m.MessageHandler
	m.handlers[protocol.Kick] = m.KickHandler
}

func (m *Manager) HandshakeHandler(packet *protocol.Packet, c Connection) error {
	res := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Heartbeat: 3,
		},
	}
	data, _ := json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		zap.L().Error("encode packet err: ", zap.Error(err))
		return err
	}
	return c.SendMessage(buf)
}

func (m *Manager) HandshakeAckHandler(packet *protocol.Packet, c Connection) error {
	zap.L().Info("receiver handshake ack message...")
	return nil
}

func (m *Manager) HeartbeatHandler(packet *protocol.Packet, c Connection) error {
	zap.L().Sugar().Info("receiver heartbeat message:%v", packet.Type)
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		zap.L().Error("encode packet err: ", zap.Error(err))
		return err
	}
	return c.SendMessage(buf)
}

// MessageHandler 将消息通过nats转发给对应的node
func (m *Manager) MessageHandler(packet *protocol.Packet, c Connection) error {
	message := packet.MessageBody()
	zap.L().Sugar().Infof("receiver message body, type=%v, router=%v, data:%v",
		message.Type, message.Route, string(message.Data))
	// connector.entryHandler.entry
	routeStr := message.Route
	routers := strings.Split(routeStr, ".")
	if len(routers) != 3 {
		return errors.New("router unsupported")
	}
	serverType := routers[0]
	handlerMethod := fmt.Sprintf("%s.%s", routers[1], routers[2])
	connectorConfig := game.Conf.GetConnectorByServerType(serverType)
	if connectorConfig != nil { // connectorConfig = "connector"
		// 本地connector服务器处理
		handler, ok := m.ConnectorHandlers[handlerMethod]
		if ok {
			data, err := handler(c.GetSession(), message.Data)
			if err != nil {
				return err
			}
			marshal, _ := json.Marshal(data)
			message.Type = protocol.Response
			message.Data = marshal
			encode, err := protocol.MessageEncode(message)
			if err != nil {
				return err
			}
			res, err := protocol.Encode(packet.Type, encode)
			if err != nil {
				return err
			}
			return c.SendMessage(res)
		}
	} else {
		// nats 远端调用处理 hall.userHandler.updateUserAddress
		dst, err := m.selectDst(serverType)
		if err != nil {
			zap.L().Error("remote send msg selectDst err: ", zap.Error(err))
			return err
		}
		msg := &remote.Msg{
			Cid:         c.GetSession().Cid,
			Uid:         c.GetSession().Uid,
			Src:         m.ServerId,
			Dst:         dst,
			Router:      handlerMethod,
			Body:        message,
			SessionData: c.GetSession().data, // 一个map[string]any
		}
		data, _ := json.Marshal(msg)
		err = m.RemoteCli.SendMsg(dst, data)
		if err != nil {
			zap.L().Error("remote send msg err：", zap.Error(err))
			return err
		}
	}
	return nil
}

func (m *Manager) KickHandler(packet *protocol.Packet, c Connection) error {
	zap.L().Info("receiver kick  message...")
	return nil
}

func (m *Manager) routeEvent(packet *protocol.Packet, cid string) error {
	// 根据packet.type来做不同的处理  处理器
	conn, ok := m.clients[cid]
	if ok {
		handler, ok := m.handlers[packet.Type]
		if ok {
			return handler(packet, conn)
		} else {
			return errors.New("no packetType found")
		}
	}
	return errors.New("no client found")
}

// 读取node节点通过 nats 推送的消息
func (m *Manager) remoteReadChanHandler() {
	for body := range m.RemoteReadChan {
		zap.L().Info("sub nats msg: " + string(body))
		var msg remote.Msg
		if err := json.Unmarshal(body, &msg); err != nil {
			zap.L().Error("nats remote message format err: " + err.Error())
			continue
		}

		// 需要特殊处理，session类型是存储在connection中的session 并不 推送客户端
		if msg.Type == remote.SessionType {
			m.setSessionData(msg)
			continue
		}

		if msg.Body != nil {
			if msg.Body.Type == protocol.Request || msg.Body.Type == protocol.Response {
				// 给客户端回信息 都是 response
				msg.Body.Type = protocol.Response
				m.Response(&msg)
			}

			if msg.Body.Type == protocol.Push {
				m.RemotePushChan <- &msg
			}
		}
	}
}

// 读取node节点通过 nats 推送的消息 - 服务端主动向客户端推送消息
func (m *Manager) remotePushChanHandler() {
	for body := range m.RemotePushChan {
		zap.L().Sugar().Info("nats push message:%v", body)
		if body.Body.Type == protocol.Push {
			m.Response(body)
		}
	}
}

func (m *Manager) selectDst(serverType string) (string, error) {
	serversConfigs, ok := game.Conf.ServersConf.TypeServer[serverType]
	if !ok {
		return "", errors.New("no server found")
	}
	// 随机一个 比较好的一个策略
	rand.New(rand.NewSource(time.Now().UnixNano()))
	index := rand.Intn(len(serversConfigs))
	return serversConfigs[index].ID, nil
}

func (m *Manager) setSessionData(msg remote.Msg) {
	m.RLock()
	defer m.RUnlock()
	conn, ok := m.clients[msg.Cid]
	if ok {
		conn.GetSession().SetData(msg.Uid, msg.SessionData)
	}
}

func (m *Manager) Response(msg *remote.Msg) {
	conn, ok := m.clients[msg.Cid]
	if !ok {
		zap.L().Sugar().Info("%s client down, uid = %s", msg.Cid, msg.Uid)
		return
	}

	buf, err := protocol.MessageEncode(msg.Body)
	if err != nil {
		zap.L().Sugar().Error("Response MessageEncode err:%v", zap.Error(err))
		return
	}

	res, err := protocol.Encode(protocol.Data, buf)
	if err != nil {
		zap.L().Sugar().Error("Response Encode err:%v", zap.Error(err))
		return
	}

	if msg.Body.Type == protocol.Push {
		for _, v := range m.clients {
			if utils.Contains(msg.PushUser, v.GetSession().Uid) {
				v.SendMessage(res)
			}
		}
	} else {
		conn.SendMessage(res)
	}
}

package net

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	cidBase        uint64 = 10000
	maxMessageSize int64  = 1024
	pongWait              = 10 * time.Second
	writeWait             = 10 * time.Second
	pingInterval          = (pongWait * 9) / 10
)

type WsConnection struct {
	Cid       string
	Conn      *websocket.Conn
	Manager   *Manager
	ReadChan  chan *MsgPack
	WriteChan chan []byte
	Session   *Session
}

func (w *WsConnection) Run() {
	go w.readMessage()
	go w.writeMessage()
	// 心跳检测
	w.Conn.SetPongHandler(w.PongHandler)
}

func (w *WsConnection) GetSession() *Session {
	return w.Session
}

func (w *WsConnection) SendMessage(buf []byte) error {
	w.WriteChan <- buf
	return nil
}

func (w *WsConnection) readMessage() {
	defer func() {
		w.Manager.removeClient(w)
	}()
	w.Conn.SetReadLimit(maxMessageSize)
	if err := w.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		zap.L().Error("SetReadDeadline err: ", zap.Error(err))
		return
	}
	for {
		messageType, msg, err := w.Conn.ReadMessage()
		if err != nil {
			zap.L().Error("read websocket message failed")
			break
		}
		// 只接受二进制数据
		if messageType == websocket.BinaryMessage {
			if w.ReadChan != nil {
				w.ReadChan <- &MsgPack{
					Cid:  w.Cid,
					Body: msg,
				}
			}

		} else {
			zap.L().Info("unsupported message type, messageType: ")
		}
	}

}

func (w *WsConnection) writeMessage() {
	ticker := time.NewTicker(pingInterval)
	for {
		select {
		case msg, ok := <-w.WriteChan:
			// 没有数据
			if !ok {
				if err := w.Conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					zap.L().Error("connection closed, err: ", zap.Error(err))
				}
			}
			// 读数据
			if err := w.Conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				zap.L().Sugar().Errorf("client[%s] write message err: %v", w.Cid, err)
			}
		case <-ticker.C:
			if err := w.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				zap.L().Sugar().Errorf("client[%s] ping SetWriteDeadline err :%v", w.Cid, err)
			}
			if err := w.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				zap.L().Sugar().Errorf("client[%s] ping err :%v", w.Cid, err)
			}
		}
	}

}

func (w *WsConnection) Close() {
	if w.Conn != nil {
		w.Conn.Close()
	}
}

func (w *WsConnection) PongHandler(data string) error {
	zap.L().Info("pong...")
	if err := w.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return err
	}
	return nil
}

func NewWsConnection(conn *websocket.Conn, manager *Manager) *WsConnection {
	cid := fmt.Sprintf("%s-%s-%d", uuid.NewString(), manager.ServerId, atomic.AddUint64(&cidBase, 1))
	return &WsConnection{
		Cid:       cid,
		Conn:      conn,
		Manager:   manager,
		ReadChan:  manager.ClientReadChan,
		WriteChan: make(chan []byte, 1024),
		Session:   NewSession(cid),
	}
}

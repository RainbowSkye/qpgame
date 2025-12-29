package node

import (
	"encoding/json"
	"framework/remote"

	"go.uber.org/zap"
)

type App struct {
	remoteCli remote.Client
	readChan  chan []byte
	writeChan chan *remote.Msg
	handlers  LogicHandler
}

func Default() *App {
	return &App{
		readChan:  make(chan []byte, 1024),
		writeChan: make(chan *remote.Msg, 1024),
		handlers:  make(LogicHandler),
	}
}

func (a *App) Run(serverId string) error {
	a.remoteCli = remote.NewNatsClient(serverId, a.readChan)
	err := a.remoteCli.Run()
	if err != nil {
		return err
	}
	go a.readChanMsg()
	go a.writeChanMsg()
	return nil
}

func (a *App) readChanMsg() {
	for msg := range a.readChan {
		var remoteMsg remote.Msg
		if err := json.Unmarshal(msg, &remoteMsg); err != nil {
			continue
		}

		session := remote.NewSession(a.remoteCli, &remoteMsg)
		session.SetData(remoteMsg.SessionData)

		router := remoteMsg.Router
		if handlerFunc := a.handlers[router]; handlerFunc != nil {
			result := handlerFunc(session, remoteMsg.Body.Data)

			message := remoteMsg.Body
			if result != nil {
				body, _ := json.Marshal(result)
				message.Data = body
			}

			responseMsg := &remote.Msg{
				Src:  remoteMsg.Dst,
				Dst:  remoteMsg.Src,
				Body: message,
				Uid:  remoteMsg.Uid,
				Cid:  remoteMsg.Cid,
			}

			a.writeChan <- responseMsg
		}
	}
}

func (a *App) writeChanMsg() {
	for msg := range a.writeChan {
		marshal, _ := json.Marshal(msg)
		if err := a.remoteCli.SendMsg(msg.Dst, marshal); err != nil {
			zap.L().Error("app remote send msg err: ", zap.Error(err))
		}
	}
}

func (a *App) Close() {
	if a.remoteCli != nil {
		a.remoteCli.Close()
	}
}

func (a *App) RegisterHandler(handler LogicHandler) {
	a.handlers = handler
}

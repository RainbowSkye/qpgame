package remote

import "framework/protocol"

type Msg struct {
	Cid         string
	Uid         string
	Type        int // 0 normal 1 session
	Src         string
	Dst         string
	Router      string
	Body        *protocol.Message
	SessionData map[string]any
	PushUser    []string
}

const SessionType = 1

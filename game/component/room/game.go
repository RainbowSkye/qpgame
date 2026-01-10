package room

import "framework/remote"

type GameFrame interface {
	GetGameData(session *remote.Session) any
}

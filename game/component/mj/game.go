package mj

import (
	"encoding/json"
	"fmt"
	"framework/remote"
	"game/component/base"
	"game/component/proto"
	"math/rand/v2"
	"time"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type GameFrame struct {
	r          base.RoomFrame
	gameRule   proto.GameRule
	gameData   *GameData
	logic      *Logic
	gameResult *GameResult
}

func NewGameFrame(rule proto.GameRule, r base.RoomFrame) *GameFrame {
	gameData := initGameData(rule)
	return &GameFrame{
		r:          r,
		gameRule:   rule,
		gameData:   gameData,
		logic:      NewLogic(GameType(rule.GameFrameType), rule.Qidui),
		gameResult: new(GameResult),
	}
}

func initGameData(rule proto.GameRule) *GameData {
	g := new(GameData)
	g.ChairCount = rule.MaxPlayerCount
	g.GameStatus = GameStatusNone
	g.MaxBureau = rule.Bureau
	g.CurChairID = -1
	g.HandCards = make([][]CardID, g.ChairCount)
	g.OperateArrays = make([][]OperateType, g.ChairCount)
	g.OperateRecord = make([]OperateRecord, 0)
	g.RestCardsCount = 3*9*4 + 4
	if rule.GameFrameType == HongZhong8 {
		g.RestCardsCount = 3*9*4 + 8
	}
	return g
}

func (g *GameFrame) GetGameData(session *remote.Session) any {
	curChairId := g.r.GetUsers()[session.GetUid()].ChairID
	fmt.Println("curChairID = ", curChairId)
	var gameData GameData
	copier.CopyWithOption(&gameData, g.gameData, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})

	handCards := make([][]CardID, g.gameData.ChairCount)
	for i := range g.gameData.HandCards {
		if g.gameData.HandCards[i] != nil {
			if i == curChairId {
				handCards[curChairId] = g.gameData.HandCards[i]
			} else {
				handCards[i] = make([]CardID, len(g.gameData.HandCards[i]))
				for j := range handCards[i] {
					handCards[i][j] = 36
				}
			}
		}
	}
	gameData.HandCards = handCards
	if g.gameData.GameStatus == GameStatusNone {
		gameData.RestCardsCount = 9*3*4 + 4
		if g.gameRule.GameFrameType == HongZhong8 {
			gameData.RestCardsCount = 9*3*4 + 8
		}
	}
	return gameData
}

// StartGame 开始游戏
func (g *GameFrame) StartGame(session *remote.Session, user *proto.RoomUser) {
	// 开始游戏
	g.gameData.GameStarted = true
	g.gameData.GameStatus = Dices
	// 游戏状态推送
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 30), g.getAllUsers())

	// 庄家推送
	if g.gameData.CurBureau == 0 {
		g.gameData.BankerChairID = 0
	} else {
		// TODO 赢家为庄
	}
	g.ServerMessagePush(session, GameBankerPushData(g.gameData.BankerChairID), g.getAllUsers())

	// 骰子推送
	dice1 := rand.IntN(6) + 1
	dice2 := rand.IntN(6) + 1
	g.ServerMessagePush(session, GameDicesPushData(dice1, dice2), g.getAllUsers())

	// 发牌推送
	g.sendHandCards(session)

	// 10 当前的局数推送+1
	g.gameData.CurBureau++
	g.ServerMessagePush(session, GameBureauPushData(g.gameData.CurBureau), g.getAllUsers())
}

func (g *GameFrame) GameMessageHandle(user *proto.RoomUser, session *remote.Session, msg []byte) {
	var req MessageReq
	json.Unmarshal(msg, &req)
	if req.Type == GameChatNotify {
		g.onGameChat(user, session, req.Data)
	}
}

func (g *GameFrame) ServerMessagePush(session *remote.Session, data any, users []string) {
	session.Push(users, data, "ServerMessagePush")
}

func (g *GameFrame) getAllUsers() []string {
	users := make([]string, 0, len(g.r.GetUsers()))
	for _, v := range g.r.GetUsers() {
		users = append(users, v.UserInfo.Uid)
	}
	return users
}

func (g *GameFrame) getUserByChairID(chairID int) *proto.RoomUser {
	for _, v := range g.r.GetUsers() {
		if v.ChairID == chairID {
			return v
		}
	}
	return nil
}

func (g *GameFrame) sendHandCards(session *remote.Session) {
	// 先洗牌 在发牌
	g.logic.washCards()
	for i := 0; i < g.gameData.ChairCount; i++ {
		g.gameData.HandCards[i] = g.logic.getCards(13)
		// g.gameData.HandCards[i][0] = 31
	}
	fmt.Println("pre cards num = ", g.logic.getRestCardsCount())
	for i := 0; i < g.gameData.ChairCount; i++ {
		handCards := make([][]CardID, g.gameData.ChairCount)
		for j := 0; j < g.gameData.ChairCount; j++ {
			if i == j {
				handCards[i] = g.gameData.HandCards[i]
			} else {
				handCards[j] = make([]CardID, len(g.gameData.HandCards[i]))
				for k := 0; k < len(g.gameData.HandCards[i]); k++ {
					handCards[j][k] = 36
				}
			}
		}
		// 推送牌
		uid := g.getUserByChairID(i).UserInfo.Uid
		g.ServerMessagePush(session, GameSendCardsPushData(handCards, i), []string{uid})
	}

	// 5. 剩余牌数推送
	restCardsCount := g.logic.getRestCardsCount()
	fmt.Println("pre cards num = ", restCardsCount)
	g.ServerMessagePush(session, GameRestCardsCountPushData(restCardsCount), g.getAllUsers())
	// 7. 开始游戏状态推送
	time.AfterFunc(time.Second, func() {
		g.gameData.GameStatus = Playing
		g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, GameStatusTmPlay), g.getAllUsers())
		// 玩家的操作时间了
		g.setTurn(g.gameData.BankerChairID, session)
	})
}

func (g *GameFrame) setTurn(chairID int, session *remote.Session) {
	// 8. 拿牌推送
	g.gameData.CurChairID = chairID
	fmt.Println("chairID = ", chairID)
	// 牌不能大于14
	if len(g.gameData.HandCards[chairID]) >= 14 {
		zap.L().Sugar().Warn("已经拿过牌了,chairID:%d", chairID)
		return
	}

	// 需要给所有的用户推送 这个玩家拿到了牌 给当前用户是明牌 其他人是暗牌
	card := g.logic.getCards(1)[0]
	g.gameData.HandCards[chairID] = append(g.gameData.HandCards[chairID], card)
	operateArray := g.getMyOperateArray(session, chairID, card)

	// 暗牌
	// 只能进行一次触发
	for i := 0; i < g.gameData.ChairCount; i++ {
		uid := g.getUserByChairID(i).UserInfo.Uid
		if i == chairID {
			g.ServerMessagePush(session, GameTurnPushData(chairID, int(card), OperateTime, operateArray), []string{uid})
			g.gameData.OperateArrays[i] = operateArray
			g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{
				ChairID: i,
				Card:    card,
				Operate: Get,
			})
		} else {
			g.ServerMessagePush(session, GameTurnPushData(chairID, 36, OperateTime, operateArray), []string{uid})
		}
	}

	// 9. 剩余牌数推送
	restCardsCount := g.logic.getRestCardsCount()
	g.ServerMessagePush(session, GameRestCardsCountPushData(restCardsCount), g.getAllUsers())
}

func (g *GameFrame) getMyOperateArray(session *remote.Session, id int, card CardID) []OperateType {
	var operateArray = []OperateType{Qi}
	return operateArray
}

func (g *GameFrame) onGameChat(user *proto.RoomUser, session *remote.Session, data MessageData) {
	g.ServerMessagePush(session, GameChatNotifyData(user.ChairID, data.Type, data.Msg, data.RecipientID), g.getAllUsers())
}

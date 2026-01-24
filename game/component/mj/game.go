package mj

import (
	"encoding/json"
	"fmt"
	"framework/remote"
	"game/component/base"
	"game/component/mj/mp"
	"game/component/proto"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type GameFrame struct {
	sync.Mutex
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
	g.HandCards = make([][]mp.CardID, g.ChairCount)
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

	handCards := make([][]mp.CardID, g.gameData.ChairCount)
	for i := range g.gameData.HandCards {
		if g.gameData.HandCards[i] != nil {
			if i == curChairId {
				handCards[curChairId] = g.gameData.HandCards[i]
			} else {
				handCards[i] = make([]mp.CardID, len(g.gameData.HandCards[i]))
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
	} else if req.Type == GameTurnOperateNotify {
		g.onGameTurnOperate(user, session, req.Data)
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
		handCards := make([][]mp.CardID, g.gameData.ChairCount)
		for j := 0; j < g.gameData.ChairCount; j++ {
			if i == j {
				handCards[i] = g.gameData.HandCards[i]
			} else {
				handCards[j] = make([]mp.CardID, len(g.gameData.HandCards[i]))
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

func (g *GameFrame) getMyOperateArray(session *remote.Session, id int, card mp.CardID) []OperateType {
	var operateArray = []OperateType{Qi}
	return operateArray
}

func (g *GameFrame) onGameChat(user *proto.RoomUser, session *remote.Session, data MessageData) {
	g.ServerMessagePush(session, GameChatNotifyData(user.ChairID, data.Type, data.Msg, data.RecipientID), g.getAllUsers())
}

func (g *GameFrame) onGameTurnOperate(user *proto.RoomUser, session *remote.Session, data MessageData) {
	if data.Operate == Qi { // 出牌
		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		// 删除牌
		g.gameData.HandCards[user.ChairID] = g.delCards(g.gameData.HandCards[user.ChairID], data.Card, 1)
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.gameData.OperateArrays[user.ChairID] = nil
		g.nextTurn(data.Card, session)
	} else if data.Operate == Guo { // 过
		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.setTurn(user.ChairID, session)
	} else if data.Operate == Peng { // 碰
		if data.Card == 0 {
			length := len(g.gameData.OperateRecord)
			if length == 0 {
				// 没有记录 出错了
				zap.L().Error("用户碰操作，但是没有上一个操作记录")
			} else {
				data.Card = g.gameData.OperateRecord[length-1].Card
			}
		}

		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.gameData.HandCards[user.ChairID] = g.delCards(g.gameData.HandCards[user.ChairID], data.Card, 2)
		// 碰了之后只能出牌
		g.gameData.OperateArrays[user.ChairID] = []OperateType{Qi}
		g.ServerMessagePush(session, GameTurnPushData(user.ChairID, 0, 10,
			g.gameData.OperateArrays[user.ChairID]), g.getAllUsers())
		g.gameData.CurChairID = user.ChairID
	} else if data.Operate == GangChi {
		if data.Card == 0 {
			length := len(g.gameData.OperateRecord)
			if length == 0 {
				// 没有记录 出错了
				zap.L().Error("用户吃杠操作，但是没有上一个操作记录")
			} else {
				data.Card = g.gameData.OperateRecord[length-1].Card
			}
		}

		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.gameData.HandCards[user.ChairID] = g.delCards(g.gameData.HandCards[user.ChairID], data.Card, 3)
		// 杠了之后只能出牌
		g.gameData.OperateArrays[user.ChairID] = []OperateType{Qi}
		g.ServerMessagePush(session, GameTurnPushData(user.ChairID, 0, 10,
			g.gameData.OperateArrays[user.ChairID]), g.getAllUsers())
		g.gameData.CurChairID = user.ChairID
	} else if data.Operate == HuChi { // 吃胡
		if data.Card == 0 {
			length := len(g.gameData.OperateRecord)
			if length == 0 {
				// 没有记录 出错了
				zap.L().Error("用户吃胡操作，但是没有上一个操作记录")
			} else {
				data.Card = g.gameData.OperateRecord[length-1].Card
			}
		}

		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		g.gameData.HandCards[user.ChairID] = append(g.gameData.HandCards[user.ChairID], data.Card)
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.gameData.OperateArrays[user.ChairID] = nil
		g.gameData.CurChairID = user.ChairID
		g.gameEnd(session)
	} else if data.Operate == HuZi { // 自摸
		g.ServerMessagePush(session, GameTurnOperatePushData(user.ChairID, data.Card, data.Operate, true), g.getAllUsers())
		g.gameData.OperateRecord = append(g.gameData.OperateRecord, OperateRecord{user.ChairID, data.Card, data.Operate})
		g.gameData.OperateArrays[user.ChairID] = nil
		g.gameData.CurChairID = user.ChairID
		g.gameEnd(session)
	} else if data.Operate == GangZi { // 自摸杠

	} else if data.Operate == GangBu { // 补杠

	} else {

	}
}

func (g *GameFrame) delCards(cards []mp.CardID, card mp.CardID, times int) []mp.CardID {
	g.Lock()
	defer g.Unlock()
	newCards := make([]mp.CardID, 0)
	count := 0
	for _, v := range cards {
		if v != card {
			newCards = append(newCards, v)
		} else {
			if count == times {
				newCards = append(newCards, v)
			} else {
				count++
			}
		}
	}
	return newCards
}

func (g *GameFrame) nextTurn(card mp.CardID, session *remote.Session) {
	if card < 0 || card > 35 {
		return
	}
	// 在下一个用户出牌之前，判断一下其他用户有没有操作（碰、杠、胡等）
	hasOtherOperate := false
	for i := 0; i < g.gameData.ChairCount; i++ {
		if i == g.gameData.CurChairID {
			continue
		}

		operateArray := g.logic.getOperateArray(g.gameData.HandCards[i], card)
		if len(operateArray) > 0 {
			hasOtherOperate = true
			user := g.getUserByChairID(i)
			// todo 检查是否有bug
			g.ServerMessagePush(session, GameTurnPushData(i, int(card), 10, operateArray), []string{user.UserInfo.Uid})
			g.gameData.OperateArrays[i] = operateArray
		}
	}

	if !hasOtherOperate {
		nextTurnId := (g.gameData.CurChairID + 1) % g.gameData.ChairCount
		g.setTurn(nextTurnId, session)
	}
}

func (g *GameFrame) gameEnd(session *remote.Session) {
	g.gameData.GameStatus = Result
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 0), g.getAllUsers())
	scores := make([]int, g.gameData.ChairCount)

	l := len(g.gameData.OperateRecord)
	if l == 0 {
		zap.L().Error("没有操作记录，不可能游戏结束，请检查")
		return
	}

	lastOperate := g.gameData.OperateRecord[l-1]
	if lastOperate.Operate != HuChi || lastOperate.Operate != HuZi {
		zap.L().Error("最后一次操作，不是胡牌，不可能游戏结束，请检查")
		return
	}

	result := GameResult{
		Scores:          scores,
		HandCards:       g.gameData.HandCards,
		MyMaCards:       make([]MyMaCard, 0),
		RestCards:       g.logic.getRestCards(),
		WinChairIDArray: []int{lastOperate.ChairID},
		FangGangArray:   []int{},
		HuType:          lastOperate.Operate,
	}
	g.gameData.Result = &result
	g.ServerMessagePush(session, GameResultPushData(result), g.getAllUsers())

	time.AfterFunc(time.Second*3, func() {
		g.r.EndGame(session)
		g.resetGame(session)
	})
}

func (g *GameFrame) resetGame(session *remote.Session) {
	g.gameData.GameStarted = false
	g.gameData.GameStatus = GameStatusNone
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 0), g.getAllUsers())
	g.ServerMessagePush(session, GameRestCardsCountPushData(g.logic.getRestCardsCount()), g.getAllUsers())
	for i := 0; i < g.gameData.ChairCount; i++ {
		g.gameData.HandCards[i] = nil
		g.gameData.OperateArrays[i] = nil
	}
	g.gameData.OperateRecord = make([]OperateRecord, 0)
	g.gameData.CurChairID = -1
	g.gameData.Result = nil
}

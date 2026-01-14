package sz

import (
	"encoding/json"
	"framework/remote"
	"game/component/base"
	"game/component/proto"
	"math/rand/v2"

	"go.uber.org/zap"
)

type GameFrame struct {
	r        base.RoomFrame
	gameRule proto.GameRule
	gameData *GameData
	logic    *Logic
}

func NewGameFrame(rule proto.GameRule, r base.RoomFrame) *GameFrame {
	gameData := initGameData(rule)
	return &GameFrame{
		r:        r,
		gameRule: rule,
		gameData: gameData,
		logic:    NewLogic(),
	}
}

func initGameData(rule proto.GameRule) *GameData {
	g := &GameData{
		GameType:   GameType(rule.GameType),
		BaseScore:  rule.BaseScore,
		ChairCount: rule.MaxPlayerCount,
	}
	g.PourScores = make([][]int, g.ChairCount)
	g.HandCards = make([][]int, g.ChairCount)
	g.LookCards = make([]int, g.ChairCount)
	g.CurScores = make([]int, g.ChairCount)
	g.UserStatusArray = make([]UserStatus, g.ChairCount)
	g.UserTrustArray = []bool{false, false, false, false, false, false, false, false, false, false}
	g.Loser = make([]int, 0)
	return g
}

func (g *GameFrame) GetGameData(session *remote.Session) any {
	return g.gameData
}

func (g *GameFrame) StartGame(session *remote.Session, user *proto.RoomUser) {
	users := g.getAllUsers()
	// 1.用户信息变更推送（金币变化） {"gold": 9958, "pushRouter": 'UpdateUserInfoPush'}
	g.ServerMessagePush(session, UpdateUserInfoPushData(user.UserInfo.Gold), users)
	// 2.庄家推送 {"type":414,"data":{"bankerChairID":0},"pushRouter":"GameMessagePush"}
	if g.gameData.CurBureau == 0 { // 第一轮庄家随机，后面的轮次 霸王庄（赢的人是庄家）
		g.gameData.BankerChairID = rand.IntN(len(users))
	}
	g.gameData.CurChairID = g.gameData.BankerChairID
	g.ServerMessagePush(session, GameBankerPushData(g.gameData.BankerChairID), users)
	// 3.局数推送{"type":411,"data":{"curBureau":6},"pushRouter":"GameMessagePush"}
	g.gameData.CurBureau++
	g.ServerMessagePush(session, GameBureauPushData(g.gameData.CurBureau), users)
	// 4.游戏状态推送 分两步推送
	// 第一步 推送 发牌 牌发完之后, 第二步 推送下分 需要用户操作了 推送操作
	// {"type":401,"data":{"gameStatus":1,"tick":0},"pushRouter":"GameMessagePush"}
	g.gameData.GameStatus = SendCards
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 0), users)
	// 5.发牌推送
	g.sendCards(session)
	// 6.下分推送
	// 先推送下分状态
	g.gameData.GameStatus = PourScore
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 30), users)
	g.gameData.CurScore = g.gameRule.BaseScore * g.gameRule.AddScores[0]
	for _, v := range g.r.GetUsers() {
		g.ServerMessagePush(session, GamePourScorePushData(g.gameData.CurChairID, g.gameData.CurScore, g.gameData.CurScore,
			1, 0), []string{v.UserInfo.Uid})
	}
	// 7. 轮数推送
	g.gameData.Round = 1
	g.ServerMessagePush(session, GameRoundPushData(g.gameData.Round), users)
	// 8. 操作推送
	for _, v := range g.r.GetUsers() {
		// GameTurnPushData ChairID是做操作的座次号（是哪个用户在做操作）
		g.ServerMessagePush(session, GameTurnPushData(g.gameData.CurChairID, g.gameData.CurScore), []string{v.UserInfo.Uid})
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

func (g *GameFrame) sendCards(session *remote.Session) {
	g.logic.washCards()
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.IsPlayingChairID(i) {
			g.gameData.HandCards[i] = g.logic.getCards()
		}
	}
	// 发牌后，推送的时候，如果没有看牌的话，就是暗牌
	hands := make([][]int, g.gameData.ChairCount)
	for i, v := range g.gameData.HandCards {
		if v != nil {
			hands[i] = []int{0, 0, 0}
		}
	}
	g.ServerMessagePush(session, GameSendCardsPushData(hands), g.getAllUsers())
}

func (g *GameFrame) IsPlayingChairID(chairID int) bool {
	for _, v := range g.r.GetUsers() {
		if v.ChairID == chairID && v.UserStatus == proto.Playing {
			return true
		}
	}
	return false
}

func (g *GameFrame) GameMessageHandle(user *proto.RoomUser, session *remote.Session, msg []byte) {
	var req MessageReq
	json.Unmarshal(msg, &req)
	if req.Type == GameLookNotify { // 看牌操作
		g.onGameLook(user, session, req.Data.Cuopai)
	} else if req.Type == GamePourScoreNotify { // 加分操作
		g.onGamePourScore(user, session, req.Data.Score, req.Data.Type)
	}
}

// 看牌
func (g *GameFrame) onGameLook(user *proto.RoomUser, session *remote.Session, cuopai bool) {
	// 防御性编程
	if user.ChairID != g.gameData.CurChairID || g.gameData.GameStatus != PourScore {
		zap.L().Warn("游戏状态不是下分中或者用户席位不是游戏当前席位")
		return
	}

	if !g.IsPlayingChairID(user.ChairID) {
		zap.L().Warn("用户席位不存在")
		return
	}
	// 给所有用户推送消息
	// 当前用户看牌
	g.gameData.UserStatusArray[user.ChairID] = Look
	g.gameData.LookCards[user.ChairID] = 1
	for _, v := range g.r.GetUsers() {
		if v.ChairID == user.ChairID { // 当前用户
			g.ServerMessagePush(session, GameLookPushData(g.gameData.CurChairID,
				g.gameData.HandCards[user.ChairID], cuopai), []string{user.UserInfo.Uid})
		} else { // 其他用户
			g.ServerMessagePush(session, GameLookPushData(g.gameData.CurChairID,
				nil, cuopai), []string{user.UserInfo.Uid})
		}
	}
}

// 下分
func (g *GameFrame) onGamePourScore(user *proto.RoomUser, session *remote.Session, score int, t int) {
	// 防御性编程
	if user.ChairID != g.gameData.CurChairID || g.gameData.GameStatus != PourScore {
		zap.L().Warn("游戏状态不是下分中或者用户席位不是游戏当前席位")
		return
	}

	if !g.IsPlayingChairID(user.ChairID) {
		zap.L().Warn("用户席位不存在")
		return
	}

	if score < 0 {
		zap.L().Warn("加注或跟注分数小于0")
		return
	}

	// 更新当前用户的下分
	if g.gameData.PourScores[user.ChairID] == nil {
		g.gameData.PourScores[user.ChairID] = make([]int, 0)
	}
	g.gameData.PourScores[user.ChairID] = append(g.gameData.PourScores[user.ChairID], score)
	// 更新当前游戏总分
	g.gameData.CurScore += score
	// 更新当前用户下的总分
	g.gameData.CurScores[user.ChairID] += score
	g.ServerMessagePush(session, GamePourScorePushData(user.ChairID, score, g.gameData.CurScores[user.ChairID],
		g.gameData.CurScore, t), g.getAllUsers())
	// 2. 结束下分 座次移动到下一位 推送轮次 推送游戏状态 推送操作的座次
	g.endPourScore(session)
}

// 结束下分，座次移动，推送轮次和游戏状态
func (g *GameFrame) endPourScore(session *remote.Session) {
	// 1. 推送轮次 TODO 轮数大于规则的限制 结束游戏 进行结算
	round := g.getCurRound()
	g.ServerMessagePush(session, GameRoundPushData(round), g.getAllUsers())

	// 座次往前移动一位
	for i := 0; i < g.gameData.ChairCount; i++ {
		g.gameData.CurChairID++
		g.gameData.CurChairID = g.gameData.CurChairID % g.gameData.ChairCount
		if g.IsPlayingChairID(g.gameData.CurChairID) {
			break
		}
	}
	// 推送游戏状态
	g.gameData.GameStatus = PourScore
	g.ServerMessagePush(session, GameStatusPushData(g.gameData.GameStatus, 30), g.getAllUsers())
	// 推送操作
	g.ServerMessagePush(session, GameTurnPushData(g.gameData.CurChairID, g.gameData.CurScore), g.getAllUsers())
}

func (g *GameFrame) getCurRound() int {
	return len(g.gameData.PourScores[g.gameData.CurChairID])
}

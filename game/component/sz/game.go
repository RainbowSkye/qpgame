package sz

import (
	"common/utils"
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
		logic:      NewLogic(),
		gameResult: new(GameResult),
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
	g.Winner = make([]int, 0)
	return g
}

// GetGameData 获取游戏数据
func (g *GameFrame) GetGameData(session *remote.Session) any {
	user := g.r.GetUsers()[session.GetUid()]
	var gameData GameData
	// 深拷贝
	copier.CopyWithOption(&gameData, g.gameData, copier.Option{DeepCopy: true})
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.gameData.HandCards[i] != nil {
			gameData.HandCards[i] = make([]int, 3)
		} else {
			gameData.HandCards[i] = nil
		}
	}

	// 如果当前用户看过牌，返回牌
	if g.gameData.LookCards[user.ChairID] == 1 {
		gameData.HandCards[user.ChairID] = g.gameData.HandCards[user.ChairID]
	}

	fmt.Printf("g.gameData = %+v\n", *g.gameData)
	fmt.Printf("gameData = %+v\n", gameData)

	return gameData
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
	} else if req.Type == GameCompareNotify { // 比牌操作
		g.onGameCompare(user, session, req.Data.ChairID)
	} else if req.Type == GameAbandonNotify { // 弃牌操作
		g.onGameAbandon(user, session)
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

	// 判断剩余玩家数量(剩一名玩家)
	gamerCount := 0
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.IsPlayingChairID(i) && !utils.Contains(g.gameData.Loser, i) {
			gamerCount++
		}
	}
	if gamerCount == 1 {
		g.startResult(session)
		zap.L().Info("startResult...")
		return
	}

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

// 比牌
func (g *GameFrame) onGameCompare(curUser *proto.RoomUser, session *remote.Session, comparedChairId int) {
	if curUser.ChairID != g.gameData.CurChairID || g.gameData.GameStatus != PourScore {
		zap.L().Warn("游戏状态不是下分中或者用户席位不是游戏当前席位")
		return
	}

	if !g.IsPlayingChairID(curUser.ChairID) {
		zap.L().Warn("用户席位不存在")
		return
	}
	// 下分操作 TODO

	curCards := g.gameData.HandCards[curUser.ChairID]
	comparedCards := g.gameData.HandCards[comparedChairId]
	res := g.logic.CompareCards(curCards, comparedCards)
	if res == 0 { // 牌面大小相同，主动比牌方输
		res = -1
	}
	var winChairId int
	var loseChairId int
	if res == -1 { // 主动比牌用户输
		winChairId = comparedChairId
		loseChairId = curUser.ChairID
	} else {
		winChairId = curUser.ChairID
		loseChairId = comparedChairId
	}
	g.gameData.Loser = append(g.gameData.Loser, loseChairId)
	g.gameData.UserStatusArray[loseChairId] = Lose
	gamerCount := 0
	for i := range g.gameData.ChairCount {
		if g.gameData.HandCards[i] != nil {
			gamerCount++
		}
	}
	if len(g.gameData.Loser)+1 == gamerCount {
		zap.L().Info("win...")
		g.gameData.Winner = append(g.gameData.Winner, winChairId)
		g.gameData.UserStatusArray[loseChairId] = Win
		fmt.Println("compare len(g.gameData.Winner[0]) = ", len(g.gameData.Winner))
	}
	// 将比较结果推送给所有人
	g.ServerMessagePush(session, GameComparePushData(curUser.ChairID, loseChairId, comparedChairId, winChairId),
		g.getAllUsers())

	// 结束下分
	g.endPourScore(session)
}

// 弃牌
func (g *GameFrame) onGameAbandon(user *proto.RoomUser, session *remote.Session) {
	if user.ChairID != g.gameData.CurChairID || g.gameData.GameStatus != PourScore {
		zap.L().Warn("游戏状态不是下分中或者用户席位不是游戏当前席位")
		return
	}

	if !g.IsPlayingChairID(user.ChairID) {
		zap.L().Warn("用户席位不存在")
		return
	}
	g.gameData.UserStatusArray[user.ChairID] = Abandon
	g.gameData.Loser = append(g.gameData.Loser, user.ChairID)
	gamerCount := 0
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.IsPlayingChairID(i) && !utils.Contains(g.gameData.Loser, i) {
			gamerCount++
		}
	}
	if gamerCount == 1 {
		for i := 0; i < g.gameData.ChairCount; i++ {
			if g.IsPlayingChairID(i) && !utils.Contains(g.gameData.Loser, i) {
				g.gameData.Winner = append(g.gameData.Winner, i)
			}
		}
	}
	g.ServerMessagePush(session, GameAbandonPushData(user.ChairID, Abandon), g.getAllUsers())

	time.AfterFunc(time.Second, func() {
		g.endPourScore(session)
	})
}

// 推送游戏结果
func (g *GameFrame) startResult(session *remote.Session) {
	g.gameData.GameStatus = Result
	g.ServerMessagePush(session, GameStatusPushData(Result, 5), g.getAllUsers())

	g.gameResult.Winners = g.gameData.Winner
	g.gameResult.HandCards = g.gameData.HandCards
	g.gameResult.CurScores = g.gameData.CurScores
	g.gameResult.Losers = g.gameData.Loser

	// 计算各个玩家赢得的分数
	winScores := make([]int, g.gameData.ChairCount)
	for i := range g.gameData.ChairCount {
		if g.gameData.PourScores[i] != nil {
			scores := 0
			for _, v := range g.gameData.PourScores[i] {
				scores += v
			}
			winScores[i] -= scores
			// 只有一个赢家
			win := g.gameData.Winner[0]
			winScores[win] += scores
		}
	}

	g.gameResult.WinScores = winScores
	g.ServerMessagePush(session, GameResultPushData(g.gameResult), g.getAllUsers())
	// 重置游戏开始下一把
	g.gameEnd(session)
	g.resetGame(session)
}

// 重置游戏
func (g *GameFrame) resetGame(session *remote.Session) {
	gd := &GameData{
		GameType:   GameType(g.gameRule.GameFrameType),
		BaseScore:  g.gameRule.BaseScore,
		ChairCount: g.gameRule.MaxPlayerCount,
	}
	gd.PourScores = make([][]int, gd.ChairCount)
	gd.HandCards = make([][]int, gd.ChairCount)
	gd.LookCards = make([]int, gd.ChairCount)
	gd.CurScores = make([]int, gd.ChairCount)
	gd.UserStatusArray = make([]UserStatus, gd.ChairCount)
	gd.UserTrustArray = []bool{false, false, false, false, false, false, false, false, false, false}
	gd.Loser = make([]int, 0)
	gd.Winner = make([]int, 0)
	gd.GameStatus = GameStatus(None)
	g.gameData = gd
	g.ServerMessagePush(session, GameStatusPushData(gd.GameStatus, 0), g.getAllUsers())
	g.r.EndGame(session)
}

// 结束游戏
func (g *GameFrame) gameEnd(session *remote.Session) {
	// 赢的人当庄家
	fmt.Println("gameEnd len(g.gameData.Winner[0]) = ", len(g.gameData.Winner))
	win := g.gameData.Winner[0]
	g.gameData.BankerChairID = win
	time.AfterFunc(5*time.Second, func() {
		for _, v := range g.r.GetUsers() {
			g.r.UserReady(v.UserInfo.Uid, session)
		}
	})
}

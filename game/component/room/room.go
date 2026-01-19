package room

import (
	"core/models/entity"
	"framework/err"
	"framework/remote"
	"game/component/base"
	"game/component/proto"
	"game/component/sz"
	"game/models/request"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Room struct {
	sync.RWMutex
	Id            string
	UnionId       int64
	Union         base.UnionBase
	gameRule      proto.GameRule
	RoomCreator   *proto.RoomCreator
	users         map[string]*proto.RoomUser
	kickSchedules map[string]*time.Timer
	GameFrame     GameFrame
	isDismissed   bool
	gameStarted   bool
}

func (r *Room) GetId() string {
	return r.Id
}

func (r *Room) EndGame(session *remote.Session) {
	r.gameStarted = false
	for k := range r.users {
		r.users[k].UserStatus = proto.None
	}
}

func (r *Room) UserReady(uid string, session *remote.Session) {
	r.userReady(uid, session)
}

func NewRoom(roomId string, unionId int64, gameRule proto.GameRule, u base.UnionBase) *Room {
	r := &Room{
		Id:            roomId,
		UnionId:       unionId,
		gameRule:      gameRule,
		users:         make(map[string]*proto.RoomUser),
		Union:         u,
		kickSchedules: make(map[string]*time.Timer),
	}
	if gameRule.GameType == int(proto.PinSanZhang) {
		r.GameFrame = sz.NewGameFrame(gameRule, r)
	}
	return r
}

func (r *Room) UserEntryRoom(session *remote.Session, data *entity.User) *err.Error {
	r.RoomCreator = &proto.RoomCreator{
		Uid: data.Uid,
	}
	if r.UnionId == 1 { // 普通玩家创建
		r.RoomCreator.CreatorType = proto.UserCreatorType
	} else { // 联盟创建
		r.RoomCreator.CreatorType = proto.UnionCreatorType
	}
	// 最多6个人参加 0 - 5有6个号
	chairID := r.getEmptyChairID()
	_, ok := r.users[data.Uid]
	if !ok {
		r.users[data.Uid] = proto.ToRoomUser(data, chairID)
	}
	// 2.将房间号推送给客户端 更新数据库 当前房间号存储起来
	r.UpdateUserInfoPush(session, data.Uid)
	session.Put("roomId", r.Id)
	// 3.将游戏类型推送给客户端（用户进入游戏的推送）
	r.SelfEntryRoomPush(session, data.Uid)
	// 4.告诉其他人此用户进入房间了
	r.OtherUserEntryRoomPush(session, data.Uid)
	// 定时踢出未准备的玩家
	go r.addKickScheduleEvent(session, data.Uid)
	return nil
}

func (r *Room) UpdateUserInfoPush(session *remote.Session, uid string) {
	// {
	// 	roomID:'336842',
	// 	pushRouter:'UpdateUserInfoPush'
	// }
	pushMsg := map[string]any{
		"roomID":     r.Id,
		"pushRouter": "UpdateUserInfoPush",
	}
	// node 节点 nats client， push通过nats将消息发送给connector，connector将消息发送给客户端
	// ServerMessagePush
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func (r *Room) ServerMessagePush(session *remote.Session, data any, users []string) {
	session.Push(users, data, "ServerMessagePush")
}

func (r *Room) SelfEntryRoomPush(session *remote.Session, uid string) {
	// {
	// 	gameType: 1,
	// 	pushRouter:'SelfEntryRoomPush'
	// }
	pushMsg := map[string]any{
		"gameType":   r.gameRule.GameType,
		"pushRouter": "SelfEntryRoomPush",
	}
	// node 节点 nats client， push通过nats将消息发送给connector，connector将消息发送给客户端
	// ServerMessagePush
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func (r *Room) RoomMessageHandle(session *remote.Session, req request.RoomMessageReq) {
	if req.Type == proto.UserReadyNotify {
		r.userReady(session.GetUid(), session)
	}
	if req.Type == proto.GetRoomSceneInfoNotify {
		r.getRoomSceneInfoPush(session)
	}
}

func (r *Room) getRoomSceneInfoPush(session *remote.Session) {
	roomUserInfoArr := make([]*proto.RoomUser, 0)
	for _, v := range r.users {
		roomUserInfoArr = append(roomUserInfoArr, v)
	}
	data := map[string]any{
		"type":       proto.GetRoomSceneInfoPush,
		"pushRouter": "RoomMessagePush",
		"data": map[string]any{
			"roomID":          r.Id,
			"roomCreatorInfo": r.RoomCreator,
			"gameRule":        r.gameRule,
			"roomUserInfoArr": roomUserInfoArr,
			"gameData":        r.GameFrame.GetGameData(session),
		},
	}
	session.Push([]string{session.GetUid()}, data, "ServerMessagePush")
}

// 添加定时踢出未准备的用户 定时任务
func (r *Room) addKickScheduleEvent(session *remote.Session, uid string) {
	r.Lock()
	defer r.Unlock()
	task, ok := r.kickSchedules[uid]
	if ok {
		task.Stop()
		delete(r.kickSchedules, uid)
	}
	r.kickSchedules[uid] = time.AfterFunc(30*time.Second, func() {
		zap.L().Info("kick 定时执行，代表 用户长时间未准备,uid=" + uid)
		// 取消定时任务
		timer, ok := r.kickSchedules[uid]
		if ok {
			timer.Stop()
		}
		delete(r.kickSchedules, uid)
		// 判断用户是否需要被踢出
		user, ok := r.users[uid]
		if ok {
			// 根据用户的状态判断
			if user.UserStatus < proto.Ready {
				r.kickUser(user, session)
				// 判断是否需要解散房间（如果房间里一个人都没有的话就解散房间）
				if len(r.users) == 0 {
					r.dismissRoom()
				}
			}
		}
	})
}

// 踢出用户
func (r *Room) kickUser(user *proto.RoomUser, session *remote.Session) {
	// 将房间roomID置为空
	r.ServerMessagePush(session, proto.UpdateUserInfoPush(""), []string{user.UserInfo.Uid})
	// 通知房间内其他的所有人该用户离开房间
	users := make([]string, 0, len(r.users))
	for _, v := range r.users {
		users = append(users, v.UserInfo.Uid)
	}
	r.ServerMessagePush(session, proto.UserLeaveRoomPushData(user), users)
	// 删除该用户
	delete(r.users, user.UserInfo.Uid)
}

// 解散房间
func (r *Room) dismissRoom() {
	r.Lock()
	defer r.Unlock()
	if r.isDismissed {
		return
	}
	r.isDismissed = true
	// 取消所有的定时任务
	r.cancelAllScheduler()
	r.Union.DismissRoom(r.Id)
}

func (r *Room) cancelAllScheduler() {
	for uid, task := range r.kickSchedules {
		task.Stop()
		delete(r.kickSchedules, uid)
	}
}

// 用户准备
func (r *Room) userReady(uid string, session *remote.Session) {
	// 1.push用户的座次， 修改用户的状态，取消定时任务
	user, ok := r.users[uid]
	if !ok {
		return
	}
	user.UserStatus = proto.Ready
	task, ok := r.kickSchedules[uid]
	if ok {
		task.Stop()
		delete(r.kickSchedules, uid)
	}
	r.ServerMessagePush(session, proto.UserReadyPushData(user.ChairID), r.getAllUsers())

	// 2.判断是否开始游戏
	if r.IsStartGame() {
		r.startGame(session, user)
	}
}

func (r *Room) JoinRoom(session *remote.Session, data *entity.User) *err.Error {
	return r.UserEntryRoom(session, data)
}

// OtherUserEntryRoomPush 通知其他用户进入房间了
func (r *Room) OtherUserEntryRoomPush(session *remote.Session, uid string) {
	others := make([]string, 0, len(r.users))
	for _, v := range r.users {
		if v.UserInfo.Uid != uid {
			others = append(others, v.UserInfo.Uid)
		}
	}
	user, ok := r.users[uid]
	if ok {
		r.ServerMessagePush(session, proto.OtherUserEntryRoomPushData(user), others)
	}
}

func (r *Room) getAllUsers() []string {
	users := make([]string, 0, len(r.users))
	for _, v := range r.users {
		users = append(users, v.UserInfo.Uid)
	}
	return users
}

func (r *Room) getEmptyChairID() int {
	if len(r.users) == 0 {
		return 0
	}
	r.Lock()
	defer r.Unlock()
	chairId := 0
	for _, v := range r.users {
		if chairId == v.ChairID {
			chairId++
		}
	}
	return chairId
}

// IsStartGame 判断是否开始游戏
func (r *Room) IsStartGame() bool {
	readyUserCount := 0
	for _, v := range r.users {
		if v.UserStatus == proto.Ready {
			readyUserCount++
		}
	}
	if readyUserCount == len(r.users) && readyUserCount >= r.gameRule.MinPlayerCount {
		return true
	}
	return false
}

// 开始游戏
func (r *Room) startGame(session *remote.Session, user *proto.RoomUser) {
	if r.gameStarted {
		return
	}
	r.gameStarted = true
	// 更新房间内玩家的状态
	for _, v := range r.users {
		v.UserStatus = proto.Playing
	}
	r.GameFrame.StartGame(session, user)
}

func (r *Room) GetUsers() map[string]*proto.RoomUser {
	return r.users
}

func (r *Room) GetMessageHandle(session *remote.Session, msg []byte) {
	// 需要游戏去处理具体的消息
	user, ok := r.users[session.GetUid()]
	if !ok {
		return
	}
	r.GameFrame.GameMessageHandle(user, session, msg)
}

package biz

import (
	"errors"
	"framework/err"
)

const OK = 0

var (
	Fail                        = err.NewError(1, errors.New("请求失败"))
	RequestDataError            = err.NewError(2, errors.New("请求数据错误"))
	SqlError                    = err.NewError(3, errors.New("数据库操作错误"))
	InvalidUsers                = err.NewError(4, errors.New("无效用户"))
	PermissionNotEnough         = err.NewError(6, errors.New("权限不足"))
	SmsCodeError                = err.NewError(7, errors.New("短信验证码错误"))
	ImgCodeError                = err.NewError(8, errors.New("图形验证码错误")) // 图形验证码错误
	SmsSendFailed               = err.NewError(9, errors.New("短信发送失败"))
	ServerMaintenance           = err.NewError(10, errors.New("服务器维护"))
	NotEnoughGold               = err.NewError(11, errors.New("钻石不足"))
	UserDataLocked              = err.NewError(12, errors.New("用户数据被锁定"))
	NotEnoughScore              = err.NewError(13, errors.New("积分不足"))
	AccountOrPasswordError      = err.NewError(101, errors.New("账号或密码错误"))
	GetHallServersFail          = err.NewError(102, errors.New("获取大厅服务器失败"))
	AccountExist                = err.NewError(103, errors.New("账号已存在"))
	AccountNotExist             = err.NewError(104, errors.New("帐号不存在"))
	NotFindBindPhone            = err.NewError(105, errors.New("该手机号未绑定"))
	PhoneAlreadyBind            = err.NewError(106, errors.New("该手机号已被绑定，无法重复绑定"))
	NotFindUser                 = err.NewError(107, errors.New("用户不存在"))
	TokenInfoError              = err.NewError(201, errors.New("无效的token"))
	NotEnoughVipLevel           = err.NewError(202, errors.New("vip等级不足"))
	BlockedAccount              = err.NewError(203, errors.New("帐号已冻结"))
	AlreadyCreatedUnion         = err.NewError(204, errors.New("已经创建过牌友圈，无法重复创建"))
	UnionNotExist               = err.NewError(205, errors.New("联盟不存在"))
	UserInRoomDataLocked        = err.NewError(206, errors.New("用户在房间中，无法操作数据"))
	NotInUnion                  = err.NewError(207, errors.New("用户不在联盟中"))
	AlreadyInUnion              = err.NewError(208, errors.New("用户已经在联盟中"))
	InviteIdError               = err.NewError(209, errors.New("邀请码错误"))
	NotYourMember               = err.NewError(210, errors.New("添加的用户不是你的下级成员"))
	ForbidGiveScore             = err.NewError(211, errors.New("禁止赠送积分"))
	ForbidInviteScore           = err.NewError(212, errors.New("禁止玩家或代理邀请玩家"))
	CanNotCreateNewHongBao      = err.NewError(213, errors.New("暂时无法分发新的红包"))
	CanNotLeaveRoom             = err.NewError(305, errors.New("正在游戏中无法离开房间"))
	RoomCountReachLimit         = err.NewError(301, errors.New("房间数量到达上线"))
	LeaveRoomGoldNotEnoughLimit = err.NewError(302, errors.New("金币不足，无法开始游戏"))
	LeaveRoomGoldExceedLimit    = err.NewError(303, errors.New("金币超过最大限度，无法开始游戏"))
	NotInRoom                   = err.NewError(306, errors.New("不在该房间中"))
	RoomPlayerCountFull         = err.NewError(307, errors.New("房间玩家已满"))
	RoomNotExist                = err.NewError(308, errors.New("房间不存在"))
	CanNotEnterNotLocation      = err.NewError(309, errors.New("无法进入房间，获取定位信息失败"))
	CanNotEnterTooNear          = err.NewError(310, errors.New("无法进入房间，与房间中的其他玩家太近"))
	RedisError                  = err.NewError(311, errors.New("redis错误"))
)

package consts

import "time"

// 好友关系状态（friend.status，DR-1：双向两行，状态归属单行）
const (
	FriendStatusNormal  = 0 // 好友
	FriendStatusBlocked = 1 // 拉黑
	// FriendStatusDeleted 删除态。DR-2 物理删除方案下无写入路径，
	// 仅保留枚举完整性（与课件/项目 README 一致），friend 表查询用它做防御性过滤。
	FriendStatusDeleted = 2
)

// FriendStatusNonFriend 非好友：VO 计算值（查不到好友关系行时填充），不落库。
const FriendStatusNonFriend = -1

// 申请状态（apply_friend.status）
const (
	ApplyStatusUnread   = 0 // 未读
	ApplyStatusAccepted = 1 // 通过
	ApplyStatusRejected = 2 // 拒绝
	ApplyStatusRead     = 3 // 已读
	ApplyStatusExpired  = 4 // 过期（由过期消费者写入，归属 phase-07 调度器，DR-5）
)

// SessionTypeSingle 单聊会话类型（session.type；1 群聊 / 2 机器人，后续阶段使用）
const SessionTypeSingle = 0

// 用户账号状态（user.state）
const (
	UserStateNormal    = 0
	UserStateBanned    = 1 // 封禁：好友详情返回 403
	UserStateCancelled = 2 // 注销：对外按不存在处理
)

// 系统通知类型（system-notification-topic 消息 type，编号沿用课件）
const (
	NotifyTypeNewApply   = 101 // 收到好友申请
	NotifyTypeNewSession = 102 // 好友申请通过、生成新会话
)

// Kafka topic（DR-5：通知为提交后尽力而为；过期消费端归属 phase-07 调度器）
const (
	KafkaTopicSystemNotification    = "system-notification-topic"
	KafkaTopicFriendRequestCreation = "friend-request-creation-topic"
)

// FriendRequestTTL 好友申请有效期。
const FriendRequestTTL = 24 * time.Hour

// FriendStatusCacheKeyPrefix 阶段 4 消息校验服务的好友状态缓存 key 前缀（跨阶段契约）。
// 完整 key：{prefix}{a}:{b}，按方向读取；拉黑/取消拉黑/删除好友后必须双向清理。
const FriendStatusCacheKeyPrefix = "msg:validate:friend:status:"

// MaxApplyMessageLen 好友申请留言最大长度（字符数，DB 列 varchar(255) 字节，留出 UTF-8 余量）。
const MaxApplyMessageLen = 100

// CanTransitApplyStatus 申请状态机（不变量 2：终态不可再改）。
// 合法转换：Unread→Accepted/Rejected/Read；Read→Accepted（先已读再通过）；
// Unread/Read→Expired（仅过期消费者使用）。
func CanTransitApplyStatus(from, to int64) bool {
	switch to {
	case ApplyStatusAccepted:
		return from == ApplyStatusUnread || from == ApplyStatusRead
	case ApplyStatusRejected, ApplyStatusRead:
		return from == ApplyStatusUnread
	case ApplyStatusExpired:
		return from == ApplyStatusUnread || from == ApplyStatusRead
	default:
		return false
	}
}

// IsModifyTargetStatus 修改申请状态接口允许的目标状态（1 通过 / 2 拒绝 / 3 已读；
// 4 过期只能由过期消费者写入，不接受外部请求）。
func IsModifyTargetStatus(status int64) bool {
	return status == ApplyStatusAccepted ||
		status == ApplyStatusRejected ||
		status == ApplyStatusRead
}

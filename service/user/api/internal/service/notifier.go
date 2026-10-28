package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/service/user/api/internal/consts"
)

// sendTimeout Kafka 发送超时：通知链路不应长时间阻塞 goroutine。
const sendTimeout = 5 * time.Second

// ContactNotifier 联系人模块对外通知接口（DR-5 决策）：
// 全部为“事务提交成功后”调用，尽力而为——失败仅记日志，绝不影响业务结果。
type ContactNotifier interface {
	// NotifyNewApply 通知接收方：收到好友申请（type=101）
	NotifyNewApply(receiverId int64, senderNickname, senderAvatar, message string)
	// NotifyNewSession 通知发送方：申请已通过、新会话已生成（type=102）
	NotifyNewSession(receiverId int64, peerNickname, sessionId string)
	// RegisterFriendRequestExpiration 注册好友申请 24h 过期事件
	//（friend-request-creation-topic，消费端归属 phase-07 调度器）
	RegisterFriendRequestExpiration(applyId int64, createTime time.Time)
}

// NewContactNotifier 按 Kafka 配置构造通知器：Brokers 为空时降级为仅日志实现。
func NewContactNotifier(brokers []string) ContactNotifier {
	if len(brokers) == 0 {
		logx.Info("[kafka] 未配置 Brokers，联系人通知降级为仅日志")
		return nopNotifier{}
	}
	return &kafkaNotifier{
		notificationWriter: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  consts.KafkaTopicSystemNotification,
			Balancer:               &kafka.RoundRobin{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
		},
		expirationWriter: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  consts.KafkaTopicFriendRequestCreation,
			Balancer:               &kafka.RoundRobin{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
		},
	}
}

// notificationMessage system-notification-topic 的消息信封。
// realtime 服务（后续阶段）按 type 分派处理。
type notificationMessage struct {
	Type       int64          `json:"type"`
	ReceiverId int64          `json:"receiverId"`
	Body       map[string]any `json:"body"`
}

type kafkaNotifier struct {
	notificationWriter *kafka.Writer
	expirationWriter   *kafka.Writer
}

func (n *kafkaNotifier) NotifyNewApply(receiverId int64, senderNickname, senderAvatar, message string) {
	msg := notificationMessage{
		Type:       consts.NotifyTypeNewApply,
		ReceiverId: receiverId,
		Body: map[string]any{
			"applyNickname": senderNickname,
			"applyAvatar":   senderAvatar,
			"message":       message,
		},
	}
	n.send(n.notificationWriter, receiverId, msg)
}

func (n *kafkaNotifier) NotifyNewSession(receiverId int64, peerNickname, sessionId string) {
	msg := notificationMessage{
		Type:       consts.NotifyTypeNewSession,
		ReceiverId: receiverId,
		Body: map[string]any{
			"peerNickname": peerNickname,
			"sessionId":    sessionId,
		},
	}
	n.send(n.notificationWriter, receiverId, msg)
}

func (n *kafkaNotifier) RegisterFriendRequestExpiration(applyId int64, createTime time.Time) {
	expirationEvent := struct {
		ApplyId    int64 `json:"applyId"`
		CreateTime int64 `json:"createTime"`
		ExpireTime int64 `json:"expireTime"`
	}{CreateTime: createTime.UnixMilli(), ExpireTime: createTime.Add(consts.FriendRequestTTL).UnixMilli(), ApplyId: applyId}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	if err := n.expirationWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(applyId, 10)),
		Value: mustJSON(expirationEvent),
	}); err != nil {
		// DR-5：过期事件发送失败仅记日志；phase-07 调度器上线后由扫表兜底补偿
		logx.Errorf("[kafka] 注册好友申请过期事件失败 applyId=%d: %v", applyId, err)
		return
	}
	logx.Infof("[kafka] 注册好友申请过期事件成功 applyId=%d expireAt=%s",
		applyId, createTime.Add(consts.FriendRequestTTL).Format(time.DateTime))
}

func (n *kafkaNotifier) send(w *kafka.Writer, receiverId int64, msg notificationMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(receiverId, 10)),
		Value: mustJSON(msg),
	}); err != nil {
		logx.Errorf("[kafka] 通知发送失败 type=%d receiverId=%d: %v", msg.Type, receiverId, err)
		return
	}
	logx.Infof("[kafka] 通知发送成功 type=%d receiverId=%d", msg.Type, receiverId)
}

// mustJSON 序列化失败时返回空对象字节（通知为尽力而为；此分支实际不可达——
// 消息结构均为静态可序列化类型）。
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// nopNotifier Kafka 未配置时的降级实现：仅记录日志，保持调用方逻辑不变。
type nopNotifier struct{}

func (nopNotifier) NotifyNewApply(receiverId int64, senderNickname, senderAvatar, message string) {
	logx.Infof("[notify:nop] 收到好友申请 receiverId=%d sender=%s", receiverId, senderNickname)
}

func (nopNotifier) NotifyNewSession(receiverId int64, peerNickname, sessionId string) {
	logx.Infof("[notify:nop] 好友申请通过 receiverId=%d peer=%s sessionId=%s", receiverId, peerNickname, sessionId)
}

func (nopNotifier) RegisterFriendRequestExpiration(applyId int64, createTime time.Time) {
	logx.Infof("[notify:nop] 注册过期事件 applyId=%d expireAt=%s",
		applyId, createTime.Add(consts.FriendRequestTTL).Format(time.DateTime))
}

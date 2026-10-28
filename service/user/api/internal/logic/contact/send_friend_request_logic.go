// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
	applyfriendmodel "go-chat/service/user/model/apply_friend"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendFriendRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendFriendRequestLogic {
	return &SendFriendRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendFriendRequest 发送好友申请。
// 不变量 1 幂等：存在同向待处理申请（Unread/Read）时直接返回原申请 ID，不重复累计；
// 被拒绝/已过期的申请允许重新发起新申请。
func (l *SendFriendRequestLogic) SendFriendRequest(req *types.SendFriendRequestReq) (*types.SendFriendRequestResp, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if req.ReceiveuserId == userId {
		return nil, resultParam("不能添加自己为好友")
	}

	// 目标用户必须存在且状态正常（封禁对外可见、注销按不存在处理）
	target, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ReceiveuserId)
	if err != nil {
		if isNotFound(err) {
			return nil, resultNotFound("对方用户不存在")
		}
		return nil, logSystem(l, "查询对方用户失败", err)
	}
	switch target.State {
	case consts.UserStateBanned:
		return nil, resultForbidden("该用户已被封禁")
	case consts.UserStateCancelled:
		return nil, resultNotFound("对方用户不存在")
	}

	// 已是好友（本人视角关系行存在）直接拒绝
	if _, err := l.svcCtx.FriendModel.FindOneByUserIdFriendId(l.ctx, userId, req.ReceiveuserId); err == nil {
		return nil, resultConflict("你们已经是好友")
	} else if !isNotFound(err) {
		return nil, logSystem(l, "查询好友关系失败", err)
	}

	// 反向待处理申请：让对方先处理，避免双向申请竞争
	if reverse, err := l.svcCtx.ApplyFriendModel.FindOneBySenderIdReceiverId(l.ctx, req.ReceiveuserId, userId); err == nil {
		if reverse.Status == consts.ApplyStatusUnread || reverse.Status == consts.ApplyStatusRead {
			return nil, resultConflict("对方已向你发送好友申请，请到申请列表处理")
		}
	} else if !isNotFound(err) {
		return nil, logSystem(l, "查询反向申请失败", err)
	}

	// 同向申请幂等：Unread/Read 视为申请处理中，返回原申请 ID
	if exist, err := l.svcCtx.ApplyFriendModel.FindOneBySenderIdReceiverId(l.ctx, userId, req.ReceiveuserId); err == nil {
		if exist.Status == consts.ApplyStatusUnread || exist.Status == consts.ApplyStatusRead {
			l.Infof("[contact] 重复申请幂等返回 userId=%d receiveuserId=%d applyId=%d",
				userId, req.ReceiveuserId, exist.Id)
			return &types.SendFriendRequestResp{ApplyId: exist.Id}, nil
		}
	} else if !isNotFound(err) {
		return nil, logSystem(l, "查询已有申请失败", err)
	}

	applyId, err := insertApply(l, userId, req)
	if err != nil {
		return nil, err
	}

	// 事务提交后：注册 24h 过期事件 + 通知对方（尽力而为，失败仅记日志，DR-5）
	l.svcCtx.Notifier.RegisterFriendRequestExpiration(applyId, time.Now())
	sender, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err == nil {
		l.svcCtx.Notifier.NotifyNewApply(req.ReceiveuserId, sender.Nickname, sender.Avatar, req.Message)
	} else {
		l.Errorf("[contact] 通知前查询申请人信息失败 userId=%d: %v", userId, err)
	}
	return &types.SendFriendRequestResp{ApplyId: applyId}, nil
}

func insertApply(l *SendFriendRequestLogic, userId int64, req *types.SendFriendRequestReq) (int64, error) {
	applyId, err := nextId()
	if err != nil {
		return 0, logSystem(l, "生成申请ID失败", err)
	}
	err = l.svcCtx.Conn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		return l.svcCtx.ApplyFriendModel.TxInsert(ctx, session, &applyfriendmodel.ApplyFriend{
			Id:         applyId,
			SenderId:   userId,
			ReceiverId: req.ReceiveuserId,
			Message:    req.Message,
			Status:     consts.ApplyStatusUnread,
		})
	})
	if err != nil {
		return 0, logSystem(l, "保存好友申请失败", err)
	}
	// TxInsert 不走 sqlc 缓存，提交后必须清理 sender:receiver 唯一索引缓存，
	// 否则后续按该索引的读取会命中插入前的 NotFound 占位（model 头部约定）
	if err := l.svcCtx.ApplyFriendModel.EvictCache(l.ctx, &applyfriendmodel.ApplyFriend{
		Id: applyId, SenderId: userId, ReceiverId: req.ReceiveuserId,
	}); err != nil {
		l.Errorf("[contact] 清理申请缓存失败 applyId=%d: %v", applyId, err)
	}
	return applyId, nil
}

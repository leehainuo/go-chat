// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

type ModifyFriendApplicationStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewModifyFriendApplicationStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ModifyFriendApplicationStatusLogic {
	return &ModifyFriendApplicationStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// fromStatuses 目标状态的合法来源状态集合（即申请状态机，不变量 2）。
// 通过/拒绝：Unread、Read（对方撤回提示语义）均可流转；标记已读：仅 Unread。
func fromStatuses(to int64) []int64 {
	switch to {
	case consts.ApplyStatusAccepted, consts.ApplyStatusRejected:
		return []int64{consts.ApplyStatusUnread, consts.ApplyStatusRead}
	case consts.ApplyStatusRead:
		return []int64{consts.ApplyStatusUnread}
	default:
		return nil
	}
}

// ModifyFriendApplicationStatus 批量修改我收到的申请状态。
// 不变量 2：条件更新（仅 from 状态命中才生效）保证并发下终态不可再改；
// 不变量 3：通过申请时好友关系 + 单聊会话与状态更新在同一事务（AddFriendTx）。
// 单条被并发修改/不存在时幂等跳过，不阻断同批其他申请。
func (l *ModifyFriendApplicationStatusLogic) ModifyFriendApplicationStatus(req *types.ModifyFriendApplicationStatusReq) (*types.ModifyApplyStatusResp, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	froms := fromStatuses(req.Status)
	if len(froms) == 0 {
		return nil, resultParam("非法的申请状态")
	}
	if len(req.SenderIds) == 0 {
		return nil, resultParam("senderIds 不能为空")
	}

	var lastSessionId string
	for _, senderId := range req.SenderIds {
		row, err := l.svcCtx.ApplyFriendModel.FindOneBySenderIdReceiverId(l.ctx, senderId, userId)
		if err != nil {
			if isNotFound(err) {
				l.Infof("[contact] 申请不存在，跳过 receiverId=%d senderId=%d", userId, senderId)
				continue
			}
			return nil, logSystem(l, "查询申请记录失败", err)
		}

		var sessionId int64
		err = l.svcCtx.Conn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
			rows, err := l.svcCtx.ApplyFriendModel.TxUpdateStatusById(ctx, session, row.Id, req.Status, froms)
			if err != nil {
				return err
			}
			if rows == 0 {
				// 状态已被并发修改（或已处于终态），按幂等跳过
				l.Infof("[contact] 申请状态已变化，跳过 applyId=%d to=%d", row.Id, req.Status)
				return nil
			}
			if req.Status != consts.ApplyStatusAccepted {
				return nil
			}
			// 不变量 3：通过申请 → 好友关系（双向）+ 单聊会话 + 双向会话关联，同一事务
			sessionId, err = l.svcCtx.Friends.AddFriendTx(ctx, session, senderId, userId)
			return err
		})
		if err != nil {
			return nil, logSystem(l, "修改申请状态失败", err)
		}
		if sessionId > 0 {
			lastSessionId = int64ToStr(sessionId)
			// 好友关系走 TxInsert 不经 sqlc，提交后清理双向唯一索引缓存；
			// 申请行状态经 TxUpdateStatusById 变更，同样需清理行缓存
			if err := l.svcCtx.FriendModel.EvictCacheByBothDirections(l.ctx, senderId, userId); err != nil {
				l.Errorf("[contact] 清理好友关系缓存失败 userId=%d friendId=%d: %v", userId, senderId, err)
			}
			if err := l.svcCtx.ApplyFriendModel.EvictCache(l.ctx, row); err != nil {
				l.Errorf("[contact] 清理申请缓存失败 applyId=%d: %v", row.Id, err)
			}
			// 提交后通知对方（尽力而为，DR-5）
			me, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
			if err == nil {
				l.svcCtx.Notifier.NotifyNewSession(senderId, me.Nickname, lastSessionId)
			} else {
				l.Errorf("[contact] 通知前查询本人信息失败 userId=%d: %v", userId, err)
			}
		}
	}
	return &types.ModifyApplyStatusResp{SessionId: lastSessionId}, nil
}

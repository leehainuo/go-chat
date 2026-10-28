// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadApplyCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUnreadApplyCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadApplyCountLogic {
	return &GetUnreadApplyCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetUnreadApplyCount 我是接收方且状态为 Unread 的申请数量。
func (l *GetUnreadApplyCountLogic) GetUnreadApplyCount(req *types.GetUnreadApplyCountReq) (*types.GetUnreadApplyCountResp, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	count, err := l.svcCtx.ApplyFriendModel.CountByReceiverAndStatus(l.ctx, userId, consts.ApplyStatusUnread)
	if err != nil {
		return nil, logSystem(l, "查询未读申请数失败", err)
	}
	return &types.GetUnreadApplyCountResp{Count: count}, nil
}

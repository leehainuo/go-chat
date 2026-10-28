// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFriendDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendDetailLogic {
	return &GetFriendDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetFriendDetail 好友详情：账号状态校验 + 用户信息 + 共同单聊会话 + 好友状态。
func (l *GetFriendDetailLogic) GetFriendDetail(req *types.GetFriendDetailReq) (*types.FriendDetailVO, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Friends.GetFriendDetail(l.ctx, userId, req.FriendId)
}

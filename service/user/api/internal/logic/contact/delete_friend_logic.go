// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFriendLogic {
	return &DeleteFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteFriend 删除好友（DR-2 物理删，双向）。
func (l *DeleteFriendLogic) DeleteFriend(req *types.UserIdAndReceiveuserIdReq) error {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return err
	}
	return l.svcCtx.Friends.DeleteFriend(l.ctx, userId, req.ReceiveuserId)
}

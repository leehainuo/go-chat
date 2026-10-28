// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnblockFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnblockFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnblockFriendLogic {
	return &UnblockFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UnblockFriend 取消拉黑（仅当当前为拉黑态，幂等保护）。
func (l *UnblockFriendLogic) UnblockFriend(req *types.UserIdAndReceiveuserIdReq) error {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return err
	}
	return l.svcCtx.Friends.UnblockFriend(l.ctx, userId, req.ReceiveuserId)
}

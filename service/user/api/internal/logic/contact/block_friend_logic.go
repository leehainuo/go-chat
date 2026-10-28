// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BlockFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBlockFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BlockFriendLogic {
	return &BlockFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// BlockFriend 拉黑好友（DR-1 单向语义，幂等）。
func (l *BlockFriendLogic) BlockFriend(req *types.UserIdAndReceiveuserIdReq) error {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return err
	}
	return l.svcCtx.Friends.BlockFriend(l.ctx, userId, req.ReceiveuserId)
}

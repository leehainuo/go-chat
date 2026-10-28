// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFriendsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendsLogic {
	return &GetFriendsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetFriends 好友列表：内存过滤 + 内存分页。
func (l *GetFriendsLogic) GetFriends(req *types.GetFriendsReq) (*types.GetFriendsResp, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Friends.GetFriends(l.ctx, userId, req.PageNum, req.PageSize, req.Keyword)
}

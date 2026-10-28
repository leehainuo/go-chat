// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUserLogic {
	return &SearchUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SearchUser 按手机号/邮箱精确搜索单个用户（只能搜索本人视角，DR-6）。
func (l *SearchUserLogic) SearchUser(req *types.SearchUserReq) (*types.FriendDetailVO, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return l.svcCtx.Friends.SearchUser(l.ctx, userId, req.Keyword)
}

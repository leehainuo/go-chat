package user

import (
	"context"

	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Logout 退出：删除 accessToken 白名单、refreshToken 白名单、单点会话 key 与 wsServerUri 路由。
func (l *LogoutLogic) Logout() (*types.SimpleResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Tokens.Logout(l.ctx, userId, mustAccessToken(l.ctx)); err != nil {
		return nil, err
	}
	return &types.SimpleResp{Ok: true}, nil
}

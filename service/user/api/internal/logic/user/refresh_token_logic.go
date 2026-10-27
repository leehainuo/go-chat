package user

import (
	"context"
	"strings"

	"go-chat/pkg/result"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// RefreshToken 无感续签（DR-2 决策：轮换 refreshToken，旧 rt 立即失效）。
func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (*types.RefreshTokenResp, error) {
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		return nil, result.Param("refreshToken 不能为空", nil)
	}
	tokens, err := l.svcCtx.Tokens.RefreshTokens(l.ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return &types.RefreshTokenResp{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

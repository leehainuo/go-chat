package user

import (
	"context"

	"go-chat/pkg/result"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginCodeLogic {
	return &LoginCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginCodeLogic) LoginCode(req *types.LoginCodeReq) (*types.LoginResp, error) {
	if err := consumeCaptcha(l.ctx, l.svcCtx.Redis, req.Account, req.Code); err != nil {
		return nil, err
	}
	u, err := findByAccount(l.ctx, l.svcCtx, req.Account)
	if err != nil {
		if isNotFound(err) {
			return nil, result.NotFound("账号不存在，请先注册", nil)
		}
		return nil, result.System("查询用户失败", err)
	}
	tokens, err := l.svcCtx.Tokens.IssueTokens(l.ctx, u.UserId)
	if err != nil {
		return nil, err
	}
	return &types.LoginResp{
		UserId:       u.UserId,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		WsServerUri:  tokens.WsServerUri,
	}, nil
}

package user

import (
	"context"
	"strings"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginPwdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginPwdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginPwdLogic {
	return &LoginPwdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginPwdLogic) LoginPwd(req *types.LoginPwdReq) (*types.LoginResp, error) {
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return nil, result.Param("account 不能为空", nil)
	}
	if req.Password == "" {
		return nil, result.Param("password 不能为空", nil)
	}
	u, err := findByAccount(l.ctx, l.svcCtx, account)
	if err != nil {
		if isNotFound(err) {
			// 安全惯例：不区分“账号不存在”与“密码错误”
			return nil, result.Unauthorized("账号或密码错误", nil)
		}
		return nil, result.System("查询用户失败", err)
	}
	if utils.EncryptPassword(consts.PasswordSalt, req.Password) != u.Password {
		return nil, result.Unauthorized("账号或密码错误", nil)
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

package user

import (
	"context"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePwdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdatePwdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePwdLogic {
	return &UpdatePwdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdatePwd 修改密码：校验旧密码 → 更新 → 使 refreshToken 立即失效。
// 已签发的 accessToken 保留至自然过期（最长 30 分钟，设计卡 §4 已声明的接受窗口）。
func (l *UpdatePwdLogic) UpdatePwd(req *types.UpdatePwdReq) (*types.SimpleResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return nil, err
	}
	u, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err != nil {
		if isNotFound(err) {
			return nil, result.NotFound("用户不存在", nil)
		}
		return nil, result.System("查询用户失败", err)
	}
	if utils.EncryptPassword(consts.PasswordSalt, req.OldPassword) != u.Password {
		return nil, result.Param("旧密码错误", nil)
	}
	if err := l.svcCtx.UserModel.UpdatePasswordByUserId(l.ctx, userId,
		utils.EncryptPassword(consts.PasswordSalt, req.NewPassword)); err != nil {
		return nil, result.System("修改密码失败", err)
	}
	if err := l.svcCtx.Tokens.KickRefresh(l.ctx, userId); err != nil {
		return nil, err
	}
	return &types.SimpleResp{Ok: true}, nil
}

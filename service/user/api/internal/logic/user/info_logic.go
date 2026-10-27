package user

import (
	"context"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InfoLogic {
	return &InfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InfoLogic) Info() (*types.UserInfoResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	u, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err != nil {
		if isNotFound(err) {
			return nil, result.NotFound("用户不存在", nil)
		}
		return nil, result.System("查询用户失败", err)
	}
	// 余额账户缺失视为 0（容错：不影响资料展示）
	balance := int64(0)
	ub, err := l.svcCtx.UserBalanceModel.FindOneByUserId(l.ctx, userId)
	switch {
	case err == nil:
		balance = ub.Balance
	case isNotFound(err):
	default:
		return nil, result.System("查询余额失败", err)
	}
	return &types.UserInfoResp{
		UserId:   u.UserId,
		Nickname: u.Nickname,
		Phone:    nullToStr(u.Phone),
		Email:    nullToStr(u.Email),
		Avatar:   u.Avatar,
		Balance:  utils.FenToYuan(balance),
	}, nil
}

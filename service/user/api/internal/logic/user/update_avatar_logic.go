package user

import (
	"context"
	"strings"

	"go-chat/pkg/result"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAvatarLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAvatarLogic {
	return &UpdateAvatarLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateAvatar 更新头像：客户端先用 /uploadUrl 返回的预签名 URL 直传 MinIO，再调用本接口落库。
func (l *UpdateAvatarLogic) UpdateAvatar(req *types.UpdateAvatarReq) (*types.SimpleResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	avatar := strings.TrimSpace(req.Avatar)
	if avatar == "" {
		return nil, result.Param("avatar 不能为空", nil)
	}
	if err := l.svcCtx.UserModel.UpdateAvatarByUserId(l.ctx, userId, avatar); err != nil {
		return nil, result.System("更新头像失败", err)
	}
	return &types.SimpleResp{Ok: true}, nil
}

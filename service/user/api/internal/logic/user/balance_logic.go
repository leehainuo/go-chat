package user

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

type BalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BalanceLogic {
	return &BalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Balance 查询本人余额（DR-3 决策：限定本人，禁止越权）。
// 不变量 4：余额以“分”存储，展示层转“元”。
func (l *BalanceLogic) Balance(req *types.BalanceReq) (*types.BalanceResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId != userId {
		return nil, result.Forbidden("只能查询本人余额", nil)
	}
	ub, err := l.svcCtx.UserBalanceModel.FindOneByUserId(l.ctx, userId)
	if err != nil {
		if isNotFound(err) {
			return nil, result.NotFound("余额账户不存在", nil)
		}
		return nil, result.System("查询余额失败", err)
	}
	return &types.BalanceResp{
		UserId:     req.UserId,
		Balance:    utils.FenToYuan(ub.Balance),
		BalanceFen: ub.Balance,
	}, nil
}

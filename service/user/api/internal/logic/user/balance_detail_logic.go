package user

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

type BalanceDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBalanceDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BalanceDetailLogic {
	return &BalanceDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// BalanceDetail 本人余额流水明细分页（DR-3 决策：限定本人）。
func (l *BalanceDetailLogic) BalanceDetail(req *types.BalanceDetailReq) (*types.BalanceDetailResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId != userId {
		return nil, result.Forbidden("只能查询本人余额流水", nil)
	}
	pageNum := req.PageNum
	if pageNum < 1 {
		pageNum = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > consts.MaxPageSize {
		pageSize = consts.MaxPageSize
	}
	total, err := l.svcCtx.BalanceLogModel.CountByUserId(l.ctx, req.UserId)
	if err != nil {
		return nil, result.System("查询流水总数失败", err)
	}
	list, err := l.svcCtx.BalanceLogModel.PageByUserId(l.ctx, req.UserId, (pageNum-1)*pageSize, pageSize)
	if err != nil {
		return nil, result.System("查询流水失败", err)
	}
	items := make([]types.BalanceDetailItem, 0, len(list))
	for _, row := range list {
		items = append(items, types.BalanceDetailItem{
			LogId:        row.LogId,
			Type:         row.Type,
			TypeDesc:     balanceTypeDesc(row.Type),
			Amount:       utils.FenToYuan(row.Amount),
			BalanceAfter: utils.FenToYuan(row.BalanceAfter),
			Remark:       row.Remark,
			CreateTime:   row.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}
	pages := (total + pageSize - 1) / pageSize
	return &types.BalanceDetailResp{
		List:        items,
		Total:       total,
		PageNum:     pageNum,
		PageSize:    pageSize,
		Pages:       pages,
		HasNext:     pageNum*pageSize < total,
		HasPrevious: pageNum > 1,
	}, nil
}

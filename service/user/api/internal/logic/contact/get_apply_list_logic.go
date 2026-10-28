// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package contact

import (
	"context"

	"go-chat/service/user/api/internal/service"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
	usermodel "go-chat/service/user/model/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetApplyListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetApplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApplyListLogic {
	return &GetApplyListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetApplyList 申请列表（我收到的 + 我发出的，按更新时间倒序）。
// ApplyItem.UserId 为对方用户 ID；IsReceiver：1 我收到的申请，0 我发出的申请。
func (l *GetApplyListLogic) GetApplyList(req *types.GetApplyListReq) (*types.GetApplyListResp, error) {
	userId, err := mustSelf(l.ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := service.ClampPage(req.PageNum, req.PageSize)
	rows, total, err := l.svcCtx.ApplyFriendModel.FindPageByUserId(l.ctx, userId,
		(pageNum-1)*pageSize, pageSize)
	if err != nil {
		return nil, logSystem(l, "查询申请列表失败", err)
	}

	// 收集“对方”用户 ID（我收到的申请对方是 sender，我发出的对方是 receiver）
	peerIds := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.SenderId == userId {
			peerIds = append(peerIds, row.ReceiverId)
		} else {
			peerIds = append(peerIds, row.SenderId)
		}
	}
	peerMap := make(map[int64]*usermodel.User, len(peerIds))
	if len(peerIds) > 0 {
		users, err := l.svcCtx.UserModel.FindByUserIds(l.ctx, peerIds)
		if err != nil {
			return nil, logSystem(l, "批量查询申请方信息失败", err)
		}
		for _, u := range users {
			peerMap[u.UserId] = u
		}
	}

	list := make([]types.ApplyItem, 0, len(rows))
	for _, row := range rows {
		isReceiver := int64(0)
		peerId := row.ReceiverId
		if row.ReceiverId == userId {
			isReceiver = 1
			peerId = row.SenderId
		}
		nickname, avatar := "", ""
		if u, ok := peerMap[peerId]; ok {
			nickname, avatar = u.Nickname, u.Avatar
		}
		list = append(list, types.ApplyItem{
			UserId:     peerId,
			Nickname:   nickname,
			Avatar:     avatar,
			Message:    row.Message,
			Status:     row.Status,
			UpdateTime: formatTime(row.UpdateTime),
			IsReceiver: isReceiver,
		})
	}

	pages, hasNext, hasPrevious := service.PageMeta(total, pageNum, pageSize)
	return &types.GetApplyListResp{
		List:        list,
		Total:       total,
		PageNum:     pageNum,
		PageSize:    pageSize,
		Pages:       pages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}, nil
}

package user

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

type RefreshWsServerUriLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshWsServerUriLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshWsServerUriLogic {
	return &RefreshWsServerUriLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// RefreshWsServerUri 重新一致性哈希分配 WS 节点（扩缩容/故障切换后客户端调用）。
func (l *RefreshWsServerUriLogic) RefreshWsServerUri() (*types.WsServerUriResp, error) {
	userId, err := mustUserId(l.ctx)
	if err != nil {
		return nil, err
	}
	wsUri := l.svcCtx.Locator.GetWsServerUri(userId)
	if wsUri != "" {
		if err := l.svcCtx.Redis.Hset(consts.WsServerUriHash,
			strconv.FormatInt(userId, 10), wsUri); err != nil {
			l.Errorf("[refreshWsServerUri] 写入 wsServerUri 失败 userId=%d: %v", userId, err)
		}
	}
	return &types.WsServerUriResp{WsServerUri: wsUri}, nil
}

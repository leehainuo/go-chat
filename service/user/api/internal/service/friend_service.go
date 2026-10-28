package service

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/types"
	applyfriendmodel "go-chat/service/user/model/apply_friend"
	friendmodel "go-chat/service/user/model/friend"
	sessionmodel "go-chat/service/user/model/session"
	usermodel "go-chat/service/user/model/user"
	usersessionmodel "go-chat/service/user/model/user_session"
)

// FriendService 联系人模块好友域服务：搜索、详情、好友列表、结好/删除、拉黑。
type FriendService struct {
	UserModel        usermodel.UserModel
	ApplyFriendModel applyfriendmodel.ApplyFriendModel
	FriendModel      friendmodel.FriendModel
	SessionModel     sessionmodel.SessionModel
	UserSessionModel usersessionmodel.UserSessionModel
	Redis            *redis.Redis
	Conn             sqlx.SqlConn
}

// SearchUser 按手机号/邮箱精确搜索单个用户（DR-6：正则分派 + 等值查询，
// 手机/邮箱为唯一列，右模糊最多命中 1 条，不提供模糊列表）。
func (s *FriendService) SearchUser(ctx context.Context, userId int64, keyword string) (*types.FriendDetailVO, error) {
	if keyword == "" {
		return nil, result.Param("搜索关键字不能为空", nil)
	}
	var (
		found *usermodel.User
		err   error
	)
	switch {
	case consts.IsPhone(keyword):
		found, err = s.UserModel.FindOneByPhone(ctx, toNullString(keyword))
	case consts.IsEmail(keyword):
		found, err = s.UserModel.FindOneByEmail(ctx, toNullString(keyword))
	default:
		return nil, result.Param("请输入有效的手机号或邮箱", nil)
	}
	if err != nil {
		if isNotFoundErr(err) {
			return nil, result.NotFound("用户不存在", nil)
		}
		return nil, result.System("搜索用户失败", err)
	}
	return s.GetFriendDetail(ctx, userId, found.UserId)
}

// GetFriendDetail 好友详情：账号状态校验 + 用户信息 + 双方共同单聊会话 + 好友状态。
func (s *FriendService) GetFriendDetail(ctx context.Context, userId, friendId int64) (*types.FriendDetailVO, error) {
	friendUser, err := s.UserModel.FindOne(ctx, friendId)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, result.NotFound("用户不存在", nil)
		}
		return nil, result.System("查询用户信息失败", err)
	}
	// 账号状态（封禁对外可见、注销按不存在处理）
	switch friendUser.State {
	case consts.UserStateBanned:
		return nil, result.Forbidden("该用户已被封禁", nil)
	case consts.UserStateCancelled:
		return nil, result.NotFound("用户不存在", nil)
	}

	sessionId := s.findSingleSessionId(ctx, userId, friendId)

	status := int64(consts.FriendStatusNonFriend)
	rel, err := s.FriendModel.FindOneByUserIdFriendId(ctx, userId, friendId)
	switch {
	case err == nil:
		status = rel.Status
	case isNotFoundErr(err):
		// 非好友，保持 -1
	default:
		return nil, result.System("查询好友关系失败", err)
	}

	return &types.FriendDetailVO{
		UserId:    friendUser.UserId,
		Nickname:  friendUser.Nickname,
		Avatar:    friendUser.Avatar,
		Phone:     nullStr(friendUser.Phone),
		Email:     nullStr(friendUser.Email),
		Signature: friendUser.Description,
		Gender:    friendUser.Gender,
		Status:    status,
		SessionId: sessionId,
	}, nil
}

// GetFriends 好友列表：一次查关系行 → 批量查用户 → 一次 JOIN 查共同会话 →
// 关键字内存过滤 → 内存分页（避免课件实现的 N+1 查询）。
func (s *FriendService) GetFriends(ctx context.Context, userId, pageNum, pageSize int64, keyword string) (*types.GetFriendsResp, error) {
	pageNum, pageSize = ClampPage(pageNum, pageSize)

	relations, err := s.FriendModel.FindListByUserId(ctx, userId)
	if err != nil {
		return nil, result.System("查询好友列表失败", err)
	}
	resp := &types.GetFriendsResp{List: []types.FriendItem{}, PageNum: pageNum, PageSize: pageSize}
	if len(relations) == 0 {
		return resp, nil
	}

	friendIds := make([]int64, 0, len(relations))
	for _, rel := range relations {
		friendIds = append(friendIds, rel.FriendId)
	}
	users, err := s.UserModel.FindByUserIds(ctx, friendIds)
	if err != nil {
		return nil, result.System("批量查询好友信息失败", err)
	}
	userMap := make(map[int64]*usermodel.User, len(users))
	for _, u := range users {
		userMap[u.UserId] = u
	}
	sessionMap, err := s.UserSessionModel.FindSessionIdMapByUserAndFriendIds(ctx, userId, friendIds)
	if err != nil {
		return nil, result.System("查询好友会话失败", err)
	}

	items := make([]types.FriendItem, 0, len(relations))
	for _, rel := range relations {
		u, ok := userMap[rel.FriendId]
		if !ok {
			// 用户行缺失（理论不可达：本项目不物理删用户行），跳过防御
			continue
		}
		items = append(items, types.FriendItem{
			UserId:    u.UserId,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Signature: u.Description,
			Status:    rel.Status,
			SessionId: int64ToStr(sessionMap[rel.FriendId]),
		})
	}

	items = FilterFriendItems(items, keyword)
	pageItems, total, pages, hasNext, hasPrevious := PageSlice(items, pageNum, pageSize)
	resp.List = pageItems
	resp.Total = total
	resp.Pages = pages
	resp.HasNext = hasNext
	resp.HasPrevious = hasPrevious
	return resp, nil
}

// AddFriendTx 事务版结好：双向两行好友关系 + 单聊会话 + 双向会话关联，
// 必须在“通过好友申请”的外层事务中调用（不变量 3）。返回新建单聊会话 ID。
func (s *FriendService) AddFriendTx(ctx context.Context, session sqlx.Session, userId, friendId int64) (int64, error) {
	sessionId, err := utils.NextId()
	if err != nil {
		return 0, err
	}
	if err := s.SessionModel.TxInsert(ctx, session, &sessionmodel.Session{
		Id:   sessionId,
		Type: consts.SessionTypeSingle,
	}); err != nil {
		return 0, err
	}
	for _, uid := range []int64{userId, friendId} {
		rowId, err := utils.NextId()
		if err != nil {
			return 0, err
		}
		if err := s.UserSessionModel.TxInsert(ctx, session, &usersessionmodel.UserSession{
			Id:        rowId,
			SessionId: sessionId,
			UserId:    uid,
		}); err != nil {
			return 0, err
		}
	}
	for _, pair := range [][2]int64{{userId, friendId}, {friendId, userId}} {
		rowId, err := utils.NextId()
		if err != nil {
			return 0, err
		}
		if err := s.FriendModel.TxInsert(ctx, session, &friendmodel.Friend{
			Id:      rowId,
			UserId:  pair[0],
			FriendId: pair[1],
			Status:  consts.FriendStatusNormal,
		}); err != nil {
			return 0, err
		}
	}
	return sessionId, nil
}

// DeleteFriend 删除好友（DR-2 物理删）：事务内清申请记录（双向）、好友关系（双向）、
// 共同单聊会话及关联；提交后清 model 缓存与阶段 4 好友状态缓存（双向）。
func (s *FriendService) DeleteFriend(ctx context.Context, userId, friendId int64) error {
	rel, err := s.FriendModel.FindOneByUserIdFriendId(ctx, userId, friendId)
	if err != nil {
		if isNotFoundErr(err) {
			return result.NotFound("好友关系不存在", nil)
		}
		return result.System("查询好友关系失败", err)
	}
	// 会话在事务外读取：删除路径上会话只减不增，事务内再删（并发新建会话的场景
	// 需要“通过申请”，而申请行在同一事务内被删除，互斥）
	sessionIds, err := s.UserSessionModel.FindSingleSessionIdsBetween(ctx, userId, friendId)
	if err != nil && !isNotFoundErr(err) {
		return result.System("查询共同会话失败", err)
	}

	err = s.Conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := s.ApplyFriendModel.TxDeleteBothDirections(ctx, session, userId, friendId); err != nil {
			return err
		}
		if err := s.FriendModel.TxDeleteBothDirections(ctx, session, userId, friendId); err != nil {
			return err
		}
		if len(sessionIds) > 0 {
			if err := s.UserSessionModel.TxDeleteBySessionIds(ctx, session, sessionIds); err != nil {
				return err
			}
			if err := s.SessionModel.TxDeleteByIds(ctx, session, sessionIds); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return result.System("删除好友失败", err)
	}

	// 提交后清缓存（失败仅记日志：unique key 未清最多导致一次旧读，TTL 自愈）
	if err := s.FriendModel.EvictCache(ctx, rel); err != nil {
		logx.Errorf("[contact] 删除好友后清理缓存失败 userId=%d friendId=%d: %v", userId, friendId, err)
	}
	if err := s.FriendModel.EvictCacheByBothDirections(ctx, userId, friendId); err != nil {
		logx.Errorf("[contact] 删除好友后清理双向缓存失败 userId=%d friendId=%d: %v", userId, friendId, err)
	}
	s.evictFriendStatusCache(ctx, userId, friendId)
	return nil
}

// BlockFriend 拉黑好友：只改归属行（DR-1 单向语义），幂等——已是拉黑态直接成功。
func (s *FriendService) BlockFriend(ctx context.Context, userId, friendId int64) error {
	rel, err := s.FriendModel.FindOneByUserIdFriendId(ctx, userId, friendId)
	if err != nil {
		if isNotFoundErr(err) {
			return result.NotFound("好友关系不存在", nil)
		}
		return result.System("查询好友关系失败", err)
	}
	if rel.Status == consts.FriendStatusBlocked {
		return nil
	}
	if err := s.FriendModel.UpdateStatusByUserIdFriendId(ctx, rel, consts.FriendStatusBlocked); err != nil {
		return result.System("拉黑好友失败", err)
	}
	if err := s.FriendModel.EvictCache(ctx, rel); err != nil {
		logx.Errorf("[contact] 拉黑后清理缓存失败 userId=%d friendId=%d: %v", userId, friendId, err)
	}
	s.evictFriendStatusCache(ctx, userId, friendId)
	return nil
}

// UnblockFriend 取消拉黑：仅当当前为拉黑态（幂等保护，防误操作掩盖状态）。
func (s *FriendService) UnblockFriend(ctx context.Context, userId, friendId int64) error {
	rel, err := s.FriendModel.FindOneByUserIdFriendId(ctx, userId, friendId)
	if err != nil {
		if isNotFoundErr(err) {
			return result.NotFound("好友关系不存在", nil)
		}
		return result.System("查询好友关系失败", err)
	}
	if rel.Status != consts.FriendStatusBlocked {
		return result.Conflict("该好友未被拉黑", nil)
	}
	if err := s.FriendModel.UpdateStatusByUserIdFriendId(ctx, rel, consts.FriendStatusNormal); err != nil {
		return result.System("取消拉黑失败", err)
	}
	if err := s.FriendModel.EvictCache(ctx, rel); err != nil {
		logx.Errorf("[contact] 取消拉黑后清理缓存失败 userId=%d friendId=%d: %v", userId, friendId, err)
	}
	s.evictFriendStatusCache(ctx, userId, friendId)
	return nil
}

// evictFriendStatusCache 清理阶段 4 消息校验服务的好友状态缓存（双向两把 key，
// 跨阶段契约：msg:validate:friend:status:{a}:{b}）。
func (s *FriendService) evictFriendStatusCache(ctx context.Context, userId, friendId int64) {
	if s.Redis == nil {
		return
	}
	for _, key := range []string{
		consts.FriendStatusCacheKeyPrefix + int64ToStr(userId) + ":" + int64ToStr(friendId),
		consts.FriendStatusCacheKeyPrefix + int64ToStr(friendId) + ":" + int64ToStr(userId),
	} {
		if _, err := s.Redis.Del(key); err != nil {
			logx.Errorf("[contact] 清理好友状态缓存失败 key=%s: %v", key, err)
		}
	}
}

func (s *FriendService) findSingleSessionId(ctx context.Context, userId, friendId int64) string {
	sessionIds, err := s.UserSessionModel.FindSingleSessionIdsBetween(ctx, userId, friendId)
	if err != nil {
		if !isNotFoundErr(err) {
			logx.Errorf("[contact] 查询单聊会话失败 userId=%d friendId=%d: %v", userId, friendId, err)
		}
		return ""
	}
	if len(sessionIds) == 0 {
		return ""
	}
	return int64ToStr(sessionIds[0])
}

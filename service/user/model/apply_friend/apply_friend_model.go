package apply_friend

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ApplyFriendModel = (*customApplyFriendModel)(nil)

type (
	// ApplyFriendModel is an interface to be customized, add more methods here,
	// and implement the added methods in customApplyFriendModel.
	ApplyFriendModel interface {
		applyFriendModel
		// TxInsert 事务版插入：通过申请时申请状态与好友关系需同事务（不变量 3）
		TxInsert(ctx context.Context, session sqlx.Session, data *ApplyFriend) error
		// CountByReceiverAndStatus 未读申请数：我是接收方且状态为 Unread 的数量
		CountByReceiverAndStatus(ctx context.Context, receiverId, status int64) (int64, error)
		// FindPageByUserId 申请列表（我是发送方或接收方，按更新时间倒序分页）
		FindPageByUserId(ctx context.Context, userId int64, offset, limit int64) ([]*ApplyFriend, int64, error)
		// TxUpdateStatusById 条件更新单条申请状态：仅当当前状态 ∈ fromStatuses 时生效，
		// 返回受影响行数；RowsAffected==0 表示状态已被并发修改（不变量 2 的并发保护）
		TxUpdateStatusById(ctx context.Context, session sqlx.Session, id, toStatus int64, fromStatuses []int64) (int64, error)
		// TxUpdateStatusBySenderIdsAndReceiverId 批量条件更新（拒绝/标记已读）：Unread → toStatus
		TxUpdateStatusBySenderIdsAndReceiverId(ctx context.Context, session sqlx.Session, receiverId int64, senderIds []int64, fromStatus, toStatus int64) (int64, error)
		// TxDeleteBothDirections 事务版双向删除申请记录（删除好友，DR-2 物理删）
		TxDeleteBothDirections(ctx context.Context, session sqlx.Session, userId, friendId int64) error
		// EvictCache 清除申请行主键与唯一索引缓存 key（Tx 写入不走 ExecCtx，提交后需显式清理）
		EvictCache(ctx context.Context, data *ApplyFriend) error
	}

	customApplyFriendModel struct {
		*defaultApplyFriendModel
	}
)

// NewApplyFriendModel returns a model for the database table.
func NewApplyFriendModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ApplyFriendModel {
	return &customApplyFriendModel{
		defaultApplyFriendModel: newApplyFriendModel(conn, c, opts...),
	}
}

func (m *customApplyFriendModel) TxInsert(ctx context.Context, session sqlx.Session, data *ApplyFriend) error {
	query := "insert into " + m.table + " (" + applyFriendRowsExpectAutoSet + ") values (?, ?, ?, ?, ?)"
	_, err := session.ExecCtx(ctx, query, data.Id, data.SenderId, data.ReceiverId, data.Message, data.Status)
	return err
}

func (m *customApplyFriendModel) CountByReceiverAndStatus(ctx context.Context, receiverId, status int64) (int64, error) {
	query := "select count(*) from " + m.table + " where `receiver_id` = ? and `status` = ?"
	var count int64
	err := m.QueryRowNoCacheCtx(ctx, &count, query, receiverId, status)
	return count, err
}

func (m *customApplyFriendModel) FindPageByUserId(ctx context.Context, userId int64, offset, limit int64) ([]*ApplyFriend, int64, error) {
	where := " where `sender_id` = ? or `receiver_id` = ?"
	order := " order by `update_time` desc"
	var list []*ApplyFriend
	query := "select " + applyFriendRows + " from " + m.table + where + order + " limit ? offset ?"
	if err := m.QueryRowsNoCacheCtx(ctx, &list, query, userId, userId, limit, offset); err != nil {
		return nil, 0, err
	}
	var total int64
	countQuery := "select count(*) from " + m.table + where
	if err := m.QueryRowNoCacheCtx(ctx, &total, countQuery, userId, userId); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (m *customApplyFriendModel) TxUpdateStatusById(ctx context.Context, session sqlx.Session, id, toStatus int64, fromStatuses []int64) (int64, error) {
	if len(fromStatuses) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(fromStatuses)), ",")
	query := "update " + m.table + " set `status` = ? where `id` = ? and `status` in (" + placeholders + ")"
	args := make([]any, 0, len(fromStatuses)+2)
	args = append(args, toStatus, id)
	for _, s := range fromStatuses {
		args = append(args, s)
	}
	res, err := session.ExecCtx(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (m *customApplyFriendModel) TxUpdateStatusBySenderIdsAndReceiverId(ctx context.Context, session sqlx.Session, receiverId int64, senderIds []int64, fromStatus, toStatus int64) (int64, error) {
	if len(senderIds) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(senderIds)), ",")
	query := "update " + m.table + " set `status` = ? where `receiver_id` = ? and `sender_id` in (" +
		placeholders + ") and `status` = ?"
	args := make([]any, 0, len(senderIds)+3)
	args = append(args, toStatus, receiverId)
	for _, id := range senderIds {
		args = append(args, id)
	}
	args = append(args, fromStatus)
	res, err := session.ExecCtx(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (m *customApplyFriendModel) TxDeleteBothDirections(ctx context.Context, session sqlx.Session, userId, friendId int64) error {
	query := "delete from " + m.table +
		" where (`sender_id` = ? and `receiver_id` = ?) or (`sender_id` = ? and `receiver_id` = ?)"
	_, err := session.ExecCtx(ctx, query, userId, friendId, friendId, userId)
	return err
}

func (m *customApplyFriendModel) EvictCache(ctx context.Context, data *ApplyFriend) error {
	idKey := fmt.Sprintf("%s%v", cacheApplyFriendIdPrefix, data.Id)
	uniqueKey := fmt.Sprintf("%s%v:%v", cacheApplyFriendSenderIdReceiverIdPrefix, data.SenderId, data.ReceiverId)
	return m.DelCacheCtx(ctx, idKey, uniqueKey)
}

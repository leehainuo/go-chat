package friend

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ FriendModel = (*customFriendModel)(nil)

// statusDeleted 约束中的 Deleted=2。物理删除方案（DR-2）下无写入路径，
// 仅用于查询时的防御性过滤，避免历史数据干扰。
const statusDeleted = 2

type (
	// FriendModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFriendModel.
	FriendModel interface {
		friendModel
		// TxInsert 事务版插入：结好好友的双向两行写入需与其它表同事务（DR-1/不变量 3）
		TxInsert(ctx context.Context, session sqlx.Session, data *Friend) error
		// FindListByUserId 查询某用户的好友关系行（好友列表）
		FindListByUserId(ctx context.Context, userId int64) ([]*Friend, error)
		// UpdateStatusByUserIdFriendId 按 (user_id, friend_id) 更新状态并维护缓存（拉黑/取消拉黑，单行原子操作无需事务）
		UpdateStatusByUserIdFriendId(ctx context.Context, rel *Friend, status int64) error
		// TxDeleteBothDirections 事务版双向删除两行好友关系（删除好友，DR-2 物理删）
		TxDeleteBothDirections(ctx context.Context, session sqlx.Session, userId, friendId int64) error
		// EvictCache 清除一行关系的主键与唯一索引缓存 key（Tx 写入不走 ExecCtx，提交后需显式清理）
		EvictCache(ctx context.Context, data *Friend) error
		// EvictCacheByBothDirections 双向清理唯一索引缓存 key（行已删除、无法反查主键时使用）
		EvictCacheByBothDirections(ctx context.Context, userId, friendId int64) error
	}

	customFriendModel struct {
		*defaultFriendModel
	}
)

// NewFriendModel returns a model for the database table.
func NewFriendModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) FriendModel {
	return &customFriendModel{
		defaultFriendModel: newFriendModel(conn, c, opts...),
	}
}

func (m *customFriendModel) TxInsert(ctx context.Context, session sqlx.Session, data *Friend) error {
	query := "insert into " + m.table + " (" + friendRowsExpectAutoSet + ") values (?, ?, ?, ?)"
	_, err := session.ExecCtx(ctx, query, data.Id, data.UserId, data.FriendId, data.Status)
	return err
}

func (m *customFriendModel) FindListByUserId(ctx context.Context, userId int64) ([]*Friend, error) {
	query := "select " + friendRows + " from " + m.table +
		" where `user_id` = ? and `status` != ? order by `create_time` desc"
	var resp []*Friend
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userId, statusDeleted)
	return resp, err
}

func (m *customFriendModel) UpdateStatusByUserIdFriendId(ctx context.Context, rel *Friend, status int64) error {
	idKey := fmt.Sprintf("%s%v", cacheFriendIdPrefix, rel.Id)
	uniqueKey := fmt.Sprintf("%s%v:%v", cacheFriendUserIdFriendIdPrefix, rel.UserId, rel.FriendId)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		query := "update " + m.table + " set `status` = ? where `id` = ? and `user_id` = ? and `friend_id` = ?"
		return conn.ExecCtx(ctx, query, status, rel.Id, rel.UserId, rel.FriendId)
	}, idKey, uniqueKey)
	return err
}

func (m *customFriendModel) TxDeleteBothDirections(ctx context.Context, session sqlx.Session, userId, friendId int64) error {
	query := "delete from " + m.table +
		" where (`user_id` = ? and `friend_id` = ?) or (`user_id` = ? and `friend_id` = ?)"
	_, err := session.ExecCtx(ctx, query, userId, friendId, friendId, userId)
	return err
}

func (m *customFriendModel) EvictCache(ctx context.Context, data *Friend) error {
	idKey := fmt.Sprintf("%s%v", cacheFriendIdPrefix, data.Id)
	uniqueKey := fmt.Sprintf("%s%v:%v", cacheFriendUserIdFriendIdPrefix, data.UserId, data.FriendId)
	return m.DelCacheCtx(ctx, idKey, uniqueKey)
}

func (m *customFriendModel) EvictCacheByBothDirections(ctx context.Context, userId, friendId int64) error {
	key1 := fmt.Sprintf("%s%v:%v", cacheFriendUserIdFriendIdPrefix, userId, friendId)
	key2 := fmt.Sprintf("%s%v:%v", cacheFriendUserIdFriendIdPrefix, friendId, userId)
	return m.DelCacheCtx(ctx, key1, key2)
}

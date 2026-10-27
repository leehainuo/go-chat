package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		// TxInsert 事务版插入：注册事务（用户 + 余额 + 流水，不变量 2）中使用事务 session 连接
		TxInsert(ctx context.Context, session sqlx.Session, data *User) error
		// DeleteCache 清理该用户的三个缓存 key（主键/手机号/邮箱）。
		// TxInsert 走事务 session、不经过 ExecCtx，不会自动维护缓存；
		// 注册前的查重若未命中，go-zero 会写入 not-found 占位缓存（默认 TTL 1 分钟），
		// 事务提交后必须显式清除，否则注册后立即登录会命中占位返回 ErrNotFound（401）。
		DeleteCache(ctx context.Context, data *User) error
		// UpdatePasswordByUserId 更新密码并清除主键/邮箱/手机号三个缓存 key（否则旧数据可能从缓存读到）
		UpdatePasswordByUserId(ctx context.Context, userId int64, password string) error
		// UpdateAvatarByUserId 更新头像并清除主键缓存 key
		UpdateAvatarByUserId(ctx context.Context, userId int64, avatar string) error
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

// TxInsert 在事务 session 上执行插入；事务外的插入请使用生成的 Insert（带缓存维护）。
func (m *customUserModel) TxInsert(ctx context.Context, session sqlx.Session, data *User) error {
	query := "insert into " + m.table + " (" + userRowsExpectAutoSet + ") values (?, ?, ?, ?, ?, ?)"
	_, err := session.ExecCtx(ctx, query,
		data.UserId, data.Nickname, data.Password, data.Phone, data.Email, data.Avatar)
	return err
}

// DeleteCache 与生成的 Insert/Update 维护同一套缓存 key：
// key 用 %v 格式化 sql.NullString（形如 {13800138000 true}），查询与清除必须用同一格式才能对上。
func (m *customUserModel) DeleteCache(ctx context.Context, data *User) error {
	userEmailKey := fmt.Sprintf("%s%v", cacheUserEmailPrefix, data.Email)
	userPhoneKey := fmt.Sprintf("%s%v", cacheUserPhonePrefix, data.Phone)
	userUserIdKey := fmt.Sprintf("%s%v", cacheUserUserIdPrefix, data.UserId)
	return m.DelCacheCtx(ctx, userEmailKey, userPhoneKey, userUserIdKey)
}

func (m *customUserModel) UpdatePasswordByUserId(ctx context.Context, userId int64, password string) error {
	data, err := m.FindOne(ctx, userId)
	if err != nil {
		return err
	}
	userEmailKey := fmt.Sprintf("%s%v", cacheUserEmailPrefix, data.Email)
	userPhoneKey := fmt.Sprintf("%s%v", cacheUserPhonePrefix, data.Phone)
	userUserIdKey := fmt.Sprintf("%s%v", cacheUserUserIdPrefix, userId)
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		query := "update " + m.table + " set `password` = ? where `user_id` = ?"
		return conn.ExecCtx(ctx, query, password, userId)
	}, userEmailKey, userPhoneKey, userUserIdKey)
	return err
}

func (m *customUserModel) UpdateAvatarByUserId(ctx context.Context, userId int64, avatar string) error {
	userUserIdKey := fmt.Sprintf("%s%v", cacheUserUserIdPrefix, userId)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		query := "update " + m.table + " set `avatar` = ? where `user_id` = ?"
		return conn.ExecCtx(ctx, query, avatar, userId)
	}, userUserIdKey)
	return err
}

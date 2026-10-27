package user_balance

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserBalanceModel = (*customUserBalanceModel)(nil)

type (
	// UserBalanceModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserBalanceModel.
	UserBalanceModel interface {
		userBalanceModel
		// TxInsert 事务版插入：注册事务（用户 + 余额 + 流水，不变量 2）中使用事务 session 连接
		TxInsert(ctx context.Context, session sqlx.Session, data *UserBalance) error
	}

	customUserBalanceModel struct {
		*defaultUserBalanceModel
	}
)

// NewUserBalanceModel returns a model for the database table.
func NewUserBalanceModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserBalanceModel {
	return &customUserBalanceModel{
		defaultUserBalanceModel: newUserBalanceModel(conn, c, opts...),
	}
}

// TxInsert 在事务 session 上执行插入；事务外的插入请使用生成的 Insert（带缓存维护）。
func (m *customUserBalanceModel) TxInsert(ctx context.Context, session sqlx.Session, data *UserBalance) error {
	query := "insert into " + m.table + " (" + userBalanceRowsExpectAutoSet + ") values (?, ?, ?)"
	_, err := session.ExecCtx(ctx, query, data.BalanceId, data.UserId, data.Balance)
	return err
}

package balance_log

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ BalanceLogModel = (*customBalanceLogModel)(nil)

type (
	// BalanceLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customBalanceLogModel.
	BalanceLogModel interface {
		balanceLogModel
		// TxInsert 事务版插入：注册事务（用户 + 余额 + 流水，不变量 2）中使用事务 session 连接
		TxInsert(ctx context.Context, session sqlx.Session, data *BalanceLog) error
		// CountByUserId 流水总数（流水频繁变化，不走缓存）
		CountByUserId(ctx context.Context, userId int64) (int64, error)
		// PageByUserId 按时间倒序分页查询流水
		PageByUserId(ctx context.Context, userId int64, offset int64, limit int64) ([]*BalanceLog, error)
	}

	customBalanceLogModel struct {
		*defaultBalanceLogModel
	}
)

// NewBalanceLogModel returns a model for the database table.
func NewBalanceLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) BalanceLogModel {
	return &customBalanceLogModel{
		defaultBalanceLogModel: newBalanceLogModel(conn, c, opts...),
	}
}

// TxInsert 在事务 session 上执行插入；事务外的插入请使用生成的 Insert（带缓存维护）。
func (m *customBalanceLogModel) TxInsert(ctx context.Context, session sqlx.Session, data *BalanceLog) error {
	query := "insert into " + m.table + " (" + balanceLogRowsExpectAutoSet + ") values (?, ?, ?, ?, ?, ?)"
	_, err := session.ExecCtx(ctx, query,
		data.LogId, data.UserId, data.Type, data.Amount, data.BalanceAfter, data.Remark)
	return err
}

func (m *customBalanceLogModel) CountByUserId(ctx context.Context, userId int64) (int64, error) {
	var count int64
	err := m.QueryRowNoCacheCtx(ctx, &count,
		"select count(*) from "+m.table+" where `user_id` = ?", userId)
	return count, err
}

func (m *customBalanceLogModel) PageByUserId(ctx context.Context, userId int64, offset int64, limit int64) ([]*BalanceLog, error) {
	var list []*BalanceLog
	query := "select " + balanceLogRows + " from " + m.table +
		" where `user_id` = ? order by `create_time` desc limit ? offset ?"
	err := m.QueryRowsNoCacheCtx(ctx, &list, query, userId, limit, offset)
	return list, err
}

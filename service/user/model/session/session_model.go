package session

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SessionModel = (*customSessionModel)(nil)

type (
	// SessionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSessionModel.
	SessionModel interface {
		sessionModel
		// TxInsert 事务版插入：创建单聊会话与好友关系同事务（不变量 3）
		TxInsert(ctx context.Context, session sqlx.Session, data *Session) error
		// TxDeleteByIds 事务版按会话 ID 批量删除（删除好友，DR-2 物理删）
		TxDeleteByIds(ctx context.Context, session sqlx.Session, ids []int64) error
	}

	customSessionModel struct {
		*defaultSessionModel
	}
)

// NewSessionModel returns a model for the database table.
func NewSessionModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SessionModel {
	return &customSessionModel{
		defaultSessionModel: newSessionModel(conn, c, opts...),
	}
}

func (m *customSessionModel) TxInsert(ctx context.Context, session sqlx.Session, data *Session) error {
	query := "insert into " + m.table + " (" + sessionRowsExpectAutoSet + ") values (?, ?, ?)"
	_, err := session.ExecCtx(ctx, query, data.Id, data.Type, data.Name)
	return err
}

func (m *customSessionModel) TxDeleteByIds(ctx context.Context, session sqlx.Session, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	query := "delete from " + m.table + " where `id` in (" + placeholders + ")"
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	_, err := session.ExecCtx(ctx, query, args...)
	return err
}

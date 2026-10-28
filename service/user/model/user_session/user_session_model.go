package user_session

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserSessionModel = (*customUserSessionModel)(nil)

// SessionTypeSingle 会话类型：单聊（取值与 internal/consts 的会话枚举一致）。
const SessionTypeSingle = 0

// SessionPair 两用户共同单聊会话的行对（好友列表批量组装用）。
type SessionPair struct {
	UserId    int64 `db:"user_id"`
	SessionId int64 `db:"session_id"`
}

type (
	// UserSessionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserSessionModel.
	UserSessionModel interface {
		userSessionModel
		// TxInsert 事务版插入：创建单聊会话的两行关联与好友关系同事务（不变量 3）
		TxInsert(ctx context.Context, session sqlx.Session, data *UserSession) error
		// FindSingleSessionIdsBetween 查两用户共同的单聊会话 ID 列表
		FindSingleSessionIdsBetween(ctx context.Context, userIdA, userIdB int64) ([]int64, error)
		// FindSessionIdMapByUserAndFriendIds 一次 JOIN 查出 userId 与每个好友的共同单聊会话（好友列表批量组装，避免 N+1）
		FindSessionIdMapByUserAndFriendIds(ctx context.Context, userId int64, friendIds []int64) (map[int64]int64, error)
		// TxDeleteBySessionIds 事务版按会话 ID 批量删除关联行（删除好友，DR-2 物理删）
		TxDeleteBySessionIds(ctx context.Context, session sqlx.Session, sessionIds []int64) error
	}

	customUserSessionModel struct {
		*defaultUserSessionModel
	}
)

// NewUserSessionModel returns a model for the database table.
func NewUserSessionModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserSessionModel {
	return &customUserSessionModel{
		defaultUserSessionModel: newUserSessionModel(conn, c, opts...),
	}
}

func (m *customUserSessionModel) TxInsert(ctx context.Context, session sqlx.Session, data *UserSession) error {
	query := "insert into " + m.table + " (" + userSessionRowsExpectAutoSet + ") values (?, ?, ?)"
	_, err := session.ExecCtx(ctx, query, data.Id, data.SessionId, data.UserId)
	return err
}

func (m *customUserSessionModel) FindSingleSessionIdsBetween(ctx context.Context, userIdA, userIdB int64) ([]int64, error) {
	query := "select us1.session_id from " + m.table + " us1" +
		" inner join " + m.table + " us2 on us1.session_id = us2.session_id" +
		" inner join `session` s on us1.session_id = s.id" +
		" where us1.user_id = ? and us2.user_id = ? and s.type = ?"
	var resp []int64
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userIdA, userIdB, SessionTypeSingle)
	return resp, err
}

func (m *customUserSessionModel) FindSessionIdMapByUserAndFriendIds(ctx context.Context, userId int64, friendIds []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(friendIds))
	if len(friendIds) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(friendIds)), ",")
	query := "select us1.user_id, us1.session_id from " + m.table + " us1" +
		" inner join " + m.table + " us2 on us1.session_id = us2.session_id" +
		" inner join `session` s on us1.session_id = s.id" +
		" where us2.user_id = ? and us1.user_id in (" + placeholders + ") and s.type = ?"
	args := make([]any, 0, len(friendIds)+2)
	args = append(args, userId)
	for _, id := range friendIds {
		args = append(args, id)
	}
	args = append(args, SessionTypeSingle)
	var pairs []SessionPair
	if err := m.QueryRowsNoCacheCtx(ctx, &pairs, query, args...); err != nil {
		return nil, err
	}
	for _, p := range pairs {
		result[p.UserId] = p.SessionId
	}
	return result, nil
}

func (m *customUserSessionModel) TxDeleteBySessionIds(ctx context.Context, session sqlx.Session, sessionIds []int64) error {
	if len(sessionIds) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(sessionIds)), ",")
	query := "delete from " + m.table + " where `session_id` in (" + placeholders + ")"
	args := make([]any, 0, len(sessionIds))
	for _, id := range sessionIds {
		args = append(args, id)
	}
	_, err := session.ExecCtx(ctx, query, args...)
	return err
}

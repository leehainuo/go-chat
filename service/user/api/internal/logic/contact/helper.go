package contact

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"go-chat/pkg/ctxkey"
	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	friendmodel "go-chat/service/user/model/friend"
)

// mustUserId 从 context 取登录用户 ID（由 JWT 鉴权中间件注入）。
func mustUserId(ctx context.Context) (int64, error) {
	userId, ok := ctx.Value(ctxkey.UserId{}).(int64)
	if !ok {
		return 0, result.Unauthorized("未登录", nil)
	}
	return userId, nil
}

// mustSelf 校验路径参数 userID 与登录用户一致（防水平越权：只能操作本人视角的数据）。
func mustSelf(ctx context.Context, pathUserId int64) (int64, error) {
	userId, err := mustUserId(ctx)
	if err != nil {
		return 0, err
	}
	if pathUserId != userId {
		return 0, result.Forbidden("只能操作本人资源", nil)
	}
	return userId, nil
}

// isNotFound 统一判断“记录不存在”（go-zero model 返回 sqlx.ErrNotFound = sql.ErrNoRows）。
func isNotFound(err error) bool {
	return errors.Is(err, friendmodel.ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}

func resultParam(message string) error        { return result.Param(message, nil) }
func resultNotFound(message string) error     { return result.NotFound(message, nil) }
func resultForbidden(message string) error    { return result.Forbidden(message, nil) }
func resultConflict(message string) error     { return result.Conflict(message, nil) }

// nextId 雪花 ID 生成。
func nextId() (int64, error) {
	return utils.NextId()
}

// int64ToStr int64 转字符串（VO 中的 ID 展示）。
func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

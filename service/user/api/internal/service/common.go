package service

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-chat/service/user/api/internal/types"
)

// isNotFoundErr 统一判断“记录不存在”（go-zero model 返回 sqlx.ErrNotFound = sql.ErrNoRows）。
func isNotFoundErr(err error) bool {
	return errors.Is(err, sqlx.ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}

// toNullString 非空字符串转 sql.NullString。
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// nullStr NullString 转普通字符串（VO 组装用）。
func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// int64ToStr int64 转字符串（路径/JSON 中的 ID 展示）。
func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

// FilterFriendItems 好友列表关键字过滤：昵称 / 用户 ID 子串匹配（纯函数，便于单测）。
// 手机号未在列表项中暴露（FriendItem 无手机号字段），不做手机号匹配。
func FilterFriendItems(items []types.FriendItem, keyword string) []types.FriendItem {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return items
	}
	filtered := make([]types.FriendItem, 0, len(items))
	for _, it := range items {
		if strings.Contains(it.Nickname, keyword) || strings.Contains(int64ToStr(it.UserId), keyword) {
			filtered = append(filtered, it)
		}
	}
	return filtered
}

// PageMeta SQL 分页后的页脚元信息计算。
func PageMeta(total, pageNum, pageSize int64) (pages int64, hasNext, hasPrevious bool) {
	if total <= 0 {
		return 0, false, false
	}
	if pageSize < 1 {
		pageSize = 10
	}
	pages = (total + pageSize - 1) / pageSize
	return pages, pages > pageNum, pageNum > 1
}

// logPanic 异步任务 panic 的统一日志出口（与包内 [kafka]/[token] 日志前缀风格一致）。
// 异步任务无请求上下文，使用全局 logger。
func logPanic(taskName string, r any) {
	logx.Errorf("[panic] 异步任务发生 panic task=%s: %v", taskName, r)
}

// goSafeName 异步任务的统一 recover 包装：通知链路绝不影响主流程。
func goSafeName(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logPanic(name, r)
			}
		}()
		fn()
	}()
}

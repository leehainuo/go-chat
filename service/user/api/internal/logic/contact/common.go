package contact

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-chat/pkg/result"
)

// logSystem 统一包装：记录 error 日志并转换为 System ApiError（联系人模块专用小助手）。
func logSystem(l logx.Logger, message string, err error) error {
	l.Errorf("[contact] %s: %v", message, err)
	return result.System(message, err)
}

// formatTime 数据库时间统一展示格式。
func formatTime(t time.Time) string {
	return t.Format(time.DateTime)
}

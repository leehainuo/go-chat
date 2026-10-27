package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/redis"

	"go-chat/pkg/ctxkey"
	"go-chat/pkg/result"
	"go-chat/service/user/api/internal/consts"
	usermodel "go-chat/service/user/model/user"
	"go-chat/service/user/api/internal/svc"
)

// findByAccount 按手机/邮箱正则分派查询。
func findByAccount(ctx context.Context, svcCtx *svc.ServiceContext, account string) (*usermodel.User, error) {
	switch {
	case consts.IsPhone(account):
		return svcCtx.UserModel.FindOneByPhone(ctx, sql.NullString{String: account, Valid: true})
	case consts.IsEmail(account):
		return svcCtx.UserModel.FindOneByEmail(ctx, sql.NullString{String: account, Valid: true})
	default:
		return nil, result.Param("account 必须为合法手机号或邮箱", nil)
	}
}

// consumeCaptcha 校验并一次性消费验证码（DR-5 决策：校验通过立即删除，防止重放）。
func consumeCaptcha(ctx context.Context, rds *redis.Redis, account, code string) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return result.Param("account 不能为空", nil)
	}
	if strings.TrimSpace(code) == "" {
		return result.Param("验证码不能为空", nil)
	}
	isPhone := consts.IsPhone(account)
	if !isPhone && !consts.IsEmail(account) {
		return result.Param("account 必须为合法手机号或邮箱", nil)
	}
	key := consts.CaptchaKey(isPhone, account)
	saved, err := rds.Get(key)
	if err != nil {
		return result.System("redis error", err)
	}
	if saved == "" {
		return result.Param("验证码已过期，请重新获取", nil)
	}
	if saved != strings.TrimSpace(code) {
		return result.Param("验证码错误", nil)
	}
	if _, err := rds.Del(key); err != nil {
		return result.System("redis error", err)
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < consts.MinPasswordLen || len(password) > consts.MaxPasswordLen {
		return result.Param(fmt.Sprintf("密码长度需在 %d~%d 位之间",
			consts.MinPasswordLen, consts.MaxPasswordLen), nil)
	}
	return nil
}

// mustUserId 从 context 取登录用户 ID（由 JWT 鉴权中间件注入）。
func mustUserId(ctx context.Context) (int64, error) {
	userId, ok := ctx.Value(ctxkey.UserId{}).(int64)
	if !ok {
		return 0, result.Unauthorized("未登录", nil)
	}
	return userId, nil
}

// mustAccessToken 从 context 取原始 JWT（登出删除白名单 key 用）。
func mustAccessToken(ctx context.Context) string {
	token, _ := ctx.Value(ctxkey.AccessToken{}).(string)
	return token
}

func nullToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// isNotFound 统一判断“记录不存在”（go-zero model 返回 sqlx.ErrNotFound = sql.ErrNoRows）。
func isNotFound(err error) bool {
	return errors.Is(err, usermodel.ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}

func balanceTypeDesc(t int64) string {
	switch t {
	case consts.BalanceLogTypeInit:
		return "注册初始化"
	case 2:
		return "充值"
	case 3:
		return "扣减"
	default:
		return "其他"
	}
}

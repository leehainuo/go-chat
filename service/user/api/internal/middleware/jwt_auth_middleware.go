package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/redis"

	"go-chat/pkg/ctxkey"
	"go-chat/pkg/result"
	"go-chat/pkg/utils"
)

type JwtAuthMiddleware struct {
	Secret               string
	Redis                *redis.Redis
	AccessTokenKeyPrefix string
}

func NewJwtAuthMiddleware(secret string, rds *redis.Redis, accessTokenKeyPrefix string) *JwtAuthMiddleware {
	return &JwtAuthMiddleware{
		Secret:               secret,
		Redis:                rds,
		AccessTokenKeyPrefix: accessTokenKeyPrefix,
	}
}

// Handle 鉴权中间件（DR-3 决策）：JWT 验签 + Redis access_token:{jwt} 白名单校验。
// 白名单保证登出/踢号后 accessToken 立即失效；JWT 自身 30 分钟过期仅作兜底。
// 校验通过后将 userId 与原始 accessToken 注入 context，供 logic 与登出使用。
func (m *JwtAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			result.Write(w, r, nil, result.Unauthorized("缺少 accessToken", nil))
			return
		}
		userId, err := utils.ParseAccessToken(m.Secret, token)
		if err != nil {
			result.Write(w, r, nil, result.Unauthorized("accessToken 无效", nil))
			return
		}
		exists, err := m.Redis.Exists(m.AccessTokenKeyPrefix + token)
		if err != nil {
			result.Write(w, r, nil, result.System("鉴权服务异常", err))
			return
		}
		if !exists {
			result.Write(w, r, nil, result.Unauthorized("登录状态已失效，请重新登录", nil))
			return
		}
		ctx := context.WithValue(r.Context(), ctxkey.UserId{}, userId)
		ctx = context.WithValue(ctx, ctxkey.AccessToken{}, token)
		next(w, r.WithContext(ctx))
	}
}

func extractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if len(header) > 7 && strings.EqualFold(header[:7], "Bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

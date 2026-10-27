package ctxkey

// UserId 登录用户 ID 的 context key（由 JWT 鉴权中间件注入）。
type UserId struct{}

// AccessToken 原始 JWT 字符串的 context key（登出时需要用它删除 Redis 白名单 key）。
type AccessToken struct{}

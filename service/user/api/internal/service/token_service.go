package service

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/wslocator"
)

type Tokens struct {
	AccessToken  string
	RefreshToken string
	WsServerUri  string
}

type TokenService struct {
	redis   *redis.Redis
	secret  string
	locator *wslocator.Locator
}

func NewTokenService(rds *redis.Redis, secret string, locator *wslocator.Locator) *TokenService {
	return &TokenService{redis: rds, secret: secret, locator: locator}
}

// IssueTokens 签发双 Token 并落实单点登录（不变量 3）。
// 踢号必须先取旧 user:refresh 的值并补删 refresh_token:{旧rt}，
// 否则旧设备仍可用旧 refreshToken 续签，单点登录失效。
func (s *TokenService) IssueTokens(ctx context.Context, userId int64) (*Tokens, error) {
	// 踢掉旧会话
	oldRT, err := s.redis.Get(consts.UserRefreshKey(userId))
	if err != nil {
		return nil, result.System("redis error", err)
	}
	if oldRT != "" {
		if _, err := s.redis.Del(consts.RefreshTokenKey(oldRT)); err != nil {
			return nil, result.System("踢除旧会话失败", err)
		}
	}

	access, err := utils.GenerateAccessToken(s.secret, userId, consts.AccessTokenTTLSeconds)
	if err != nil {
		return nil, result.System("签发 accessToken 失败", err)
	}
	refresh, err := utils.NextIdStr()
	if err != nil {
		return nil, result.System("生成 refreshToken 失败", err)
	}

	userIdStr := strconv.FormatInt(userId, 10)
	if err := s.redis.Setex(consts.AccessTokenKey(access), userIdStr, consts.AccessTokenTTLSeconds); err != nil {
		return nil, result.System("写入 accessToken 失败", err)
	}
	if err := s.redis.Setex(consts.RefreshTokenKey(refresh), userIdStr, consts.RefreshTokenTTLSeconds); err != nil {
		return nil, result.System("写入 refreshToken 失败", err)
	}
	if err := s.redis.Setex(consts.UserRefreshKey(userId), refresh, consts.RefreshTokenTTLSeconds); err != nil {
		return nil, result.System("写入单点会话失败", err)
	}

	// 一致性哈希分配 WS 节点并缓存（推送服务靠它路由）；环为空时留空，不阻断登录
	wsUri := s.locator.GetWsServerUri(userId)
	if wsUri != "" {
		if err := s.redis.Hset(consts.WsServerUriHash, userIdStr, wsUri); err != nil {
			logx.Errorf("[token] 写入 wsServerUri 失败 userId=%d: %v", userId, err)
		}
	}
	return &Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		WsServerUri:  wsUri,
	}, nil
}

// RefreshTokens 无感续签续 签轮换 refreshToken
// 双重校验保证单点不变量：
//  1. refresh_token:{rt} 必须存在；
//  2. 该 rt 必须仍是 user:refresh:{userId} 的当前值——被新登录踢掉或已被轮换的旧 rt 到此直接 401。
//
// 并发续签说明：两个请求同时通过校验时，后写者覆盖 user:refresh，
// 先写者签发的新 rt 会在下次续签时被第 2 步拒绝，最终仍保持单点。
func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (*Tokens, error) {
	userIdStr, err := s.redis.Get(consts.RefreshTokenKey(refreshToken))
	if err != nil {
		return nil, result.System("redis error", err)
	}
	if userIdStr == "" {
		return nil, result.Unauthorized("refreshToken 无效或已过期，请重新登录", nil)
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return nil, result.Unauthorized("refreshToken 无效", err)
	}

	curRT, err := s.redis.Get(consts.UserRefreshKey(userId))
	if err != nil {
		return nil, result.System("redis error", err)
	}
	if curRT != refreshToken {
		return nil, result.Unauthorized("refreshToken 已失效，请重新登录", nil)
	}

	// 轮换：删除旧 rt 白名单，签发新对
	if _, err := s.redis.Del(consts.RefreshTokenKey(refreshToken)); err != nil {
		return nil, result.System("轮换 refreshToken 失败", err)
	}

	access, err := utils.GenerateAccessToken(s.secret, userId, consts.AccessTokenTTLSeconds)
	if err != nil {
		return nil, result.System("签发 accessToken 失败", err)
	}
	newRefresh, err := utils.NextIdStr()
	if err != nil {
		return nil, result.System("生成 refreshToken 失败", err)
	}

	if err := s.redis.Setex(consts.AccessTokenKey(access), userIdStr, consts.AccessTokenTTLSeconds); err != nil {
		return nil, result.System("写入 accessToken 失败", err)
	}
	if err := s.redis.Setex(consts.RefreshTokenKey(newRefresh), userIdStr, consts.RefreshTokenTTLSeconds); err != nil {
		return nil, result.System("写入 refreshToken 失败", err)
	}
	if err := s.redis.Setex(consts.UserRefreshKey(userId), newRefresh, consts.RefreshTokenTTLSeconds); err != nil {
		return nil, result.System("写入单点会话失败", err)
	}
	return &Tokens{AccessToken: access, RefreshToken: newRefresh}, nil
}

// Logout 登出：删除 accessToken 白名单、refreshToken 白名单与单点会话 key，并清除 wsServerUri 路由。
func (s *TokenService) Logout(ctx context.Context, userId int64, accessToken string) error {
	rt, err := s.redis.Get(consts.UserRefreshKey(userId))
	if err != nil {
		return result.System("redis error", err)
	}
	keys := []string{consts.AccessTokenKey(accessToken), consts.UserRefreshKey(userId)}
	if rt != "" {
		keys = append(keys, consts.RefreshTokenKey(rt))
	}
	if _, err := s.redis.Del(keys...); err != nil {
		return result.System("退出登录失败", err)
	}
	if _, err := s.redis.Hdel(consts.WsServerUriHash, strconv.FormatInt(userId, 10)); err != nil {
		logx.Errorf("[token] 删除 wsServerUri 失败 userId=%d: %v", userId, err)
	}
	return nil
}

// KickRefresh 使 userId 的 refreshToken 立即失效（修改密码后调用）。
// 已签发的 accessToken 按设计保留至自然过期（最长 30 分钟，设计卡 §4 已声明的接受风险）。
func (s *TokenService) KickRefresh(ctx context.Context, userId int64) error {
	rt, err := s.redis.Get(consts.UserRefreshKey(userId))
	if err != nil {
		return result.System("redis error", err)
	}
	keys := []string{consts.UserRefreshKey(userId)}
	if rt != "" {
		keys = append(keys, consts.RefreshTokenKey(rt))
	}
	if _, err := s.redis.Del(keys...); err != nil {
		return result.System("操作失败", err)
	}
	return nil
}

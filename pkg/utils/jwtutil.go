package utils

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateAccessToken 使用 HS256 签发 accessToken，subject 为 userId 字符串。
func GenerateAccessToken(secret string, userId int64, expireSeconds int64) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": strconv.FormatInt(userId, 10),
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken 验签并解析 accessToken，返回 userId。
func ParseAccessToken(secret, tokenStr string) (int64, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return 0, errors.New("invalid subject")
	}
	return strconv.ParseInt(sub, 10, 64)
}

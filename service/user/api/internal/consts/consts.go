package consts

import (
	"regexp"
	"strconv"
)

// 密码
const (
	PasswordSalt   = "USER" // 固定 salt，MD5 风险已在设计评审 DR-4 中确认接受
	MinPasswordLen = 6
	MaxPasswordLen = 32
)

// 验证码
const (
	CaptchaTTLSeconds = 300
	PhoneFixedCode    = "123456" // 本地开发固定码（设计卡非目标：真实短信发送）
)

// Token
const (
	AccessTokenTTLSeconds  = 30 * 60        // accessToken 30 分钟
	RefreshTokenTTLSeconds = 7 * 24 * 3600  // refreshToken 7 天
)

// Redis key
const (
	CaptchaPhonePrefix = "captcha:phone:"
	CaptchaEmailPrefix = "captcha:email:"
	AccessTokenPrefix  = "access_token:"
	RefreshTokenPrefix = "refresh_token:"
	UserRefreshPrefix  = "user:refresh:"
	WsServerUriHash    = "wsServerUri" // 在线用户 WS 节点 Hash
)

// WS
const WsChatPath = "/ws/chat"

// 余额（关键不变量：一律以“分”存储，展示层转“元”）
const (
	InitialBalanceFen  = 100000 // 注册初始化余额 1000 元
	BalanceLogTypeInit = 1      // 流水类型：注册初始化
	BalanceRemarkInit  = "注册初始化余额"
)

// MinIO
const (
	AvatarObjectPrefix = "avatar"
	PresignTTLSeconds  = 300 // 预签名 PUT URL 有效期 5 分钟（缩短写权限泄漏窗口）
	DefaultAvatar      = "/static/default-avatar.png"
)

// 分页
const MaxPageSize = 100

var (
	phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

func IsPhone(account string) bool  { return phoneRe.MatchString(account) }
func IsEmail(account string) bool  { return emailRe.MatchString(account) }

func CaptchaKey(isPhone bool, account string) string {
	if isPhone {
		return CaptchaPhonePrefix + account
	}
	return CaptchaEmailPrefix + account
}

func AccessTokenKey(token string) string { return AccessTokenPrefix + token }

func RefreshTokenKey(refreshToken string) string { return RefreshTokenPrefix + refreshToken }

func UserRefreshKey(userId int64) string {
	return UserRefreshPrefix + strconv.FormatInt(userId, 10)
}

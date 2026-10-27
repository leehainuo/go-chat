package user

import (
	"context"
	"strings"

	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCaptchaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCaptchaLogic {
	return &SendCaptchaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendCaptchaLogic) SendCaptcha(req *types.SendCaptchaReq) (*types.SimpleResp, error) {
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return nil, result.Param("account 不能为空", nil)
	}
	isPhone := consts.IsPhone(account)
	if !isPhone && !consts.IsEmail(account) {
		return nil, result.Param("account 必须为合法手机号或邮箱", nil)
	}
	// 频控：验证码 key 存在即拒绝（设计卡：重复发送拒绝）
	key := consts.CaptchaKey(isPhone, account)
	exists, err := l.svcCtx.Redis.Exists(key)
	if err != nil {
		return nil, result.System("redis error", err)
	}
	if exists {
		return nil, result.TooMany("验证码发送太频繁，请稍后再试", nil)
	}
	// 手机：本地开发固定码（设计卡非目标：真实短信）；
	// 邮箱：随机 6 位，本地经日志输出（生产环境替换为 SMTP 下发）
	code := consts.PhoneFixedCode
	if !isPhone {
		code, err = utils.RandomCaptcha(6)
		if err != nil {
			return nil, result.System("生成验证码失败", err)
		}
	}
	if err := l.svcCtx.Redis.Setex(key, code, consts.CaptchaTTLSeconds); err != nil {
		return nil, result.System("redis error", err)
	}
	l.Infof("[sendCaptcha] account=%s captcha=%s（本地开发日志输出）", account, code)
	return &types.SimpleResp{Ok: true}, nil
}

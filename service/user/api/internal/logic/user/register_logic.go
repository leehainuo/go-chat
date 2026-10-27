package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	balancelogmodel "go-chat/service/user/model/balance_log"
	usermodel "go-chat/service/user/model/user"
	userbalancemodel "go-chat/service/user/model/user_balance"
	"go-chat/pkg/result"
	"go-chat/pkg/utils"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/svc"
	"go-chat/service/user/api/internal/types"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Register 注册：验证码校验 → 唯一性校验 → 事务写用户/余额/流水（不变量 2）
// → 签发双 Token 与 wsServerUri。
// DR-5 决策：注册事务不含机器人好友，好友/会话由联系人模块负责。
func (l *RegisterLogic) Register(req *types.RegisterReq) (*types.LoginResp, error) {
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return nil, result.Param("account 不能为空", nil)
	}
	isPhone := consts.IsPhone(account)
	if !isPhone && !consts.IsEmail(account) {
		return nil, result.Param("account 必须为合法手机号或邮箱", nil)
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}
	if err := consumeCaptcha(l.ctx, l.svcCtx.Redis, account, req.Code); err != nil {
		return nil, err
	}
	// 前置查重（并发场景由唯一索引兜底，见下方 1062 映射）
	if _, err := findByAccount(l.ctx, l.svcCtx, account); err == nil {
		return nil, result.Conflict("账号已被注册", nil)
	} else if !isNotFound(err) {
		return nil, result.System("查询用户失败", err)
	}

	userId, err := utils.NextId()
	if err != nil {
		return nil, result.System("生成用户ID失败", err)
	}
	balanceId, err := utils.NextId()
	if err != nil {
		return nil, result.System("生成余额账户ID失败", err)
	}
	logId, err := utils.NextId()
	if err != nil {
		return nil, result.System("生成流水ID失败", err)
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		nickname = fmt.Sprintf("用户%d", userId%100000)
	}
	var phone, email sql.NullString
	if isPhone {
		phone = sql.NullString{String: account, Valid: true}
	} else {
		email = sql.NullString{String: account, Valid: true}
	}

	newUser := &usermodel.User{
		UserId:   userId,
		Nickname: nickname,
		Password: utils.EncryptPassword(consts.PasswordSalt, req.Password),
		Phone:    phone,
		Email:    email,
		Avatar:   consts.DefaultAvatar,
	}
	// 不变量 2：注册成功的创建用户、初始化余额同事务，要么全成要么全部回滚
	err = l.svcCtx.Conn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := l.svcCtx.UserModel.TxInsert(ctx, session, newUser); err != nil {
			return err
		}
		if err := l.svcCtx.UserBalanceModel.TxInsert(ctx, session, &userbalancemodel.UserBalance{
			BalanceId: balanceId,
			UserId:    userId,
			Balance:   consts.InitialBalanceFen,
		}); err != nil {
			return err
		}
		return l.svcCtx.BalanceLogModel.TxInsert(ctx, session, &balancelogmodel.BalanceLog{
			LogId:        logId,
			UserId:       userId,
			Type:         consts.BalanceLogTypeInit,
			Amount:       consts.InitialBalanceFen,
			BalanceAfter: consts.InitialBalanceFen,
			Remark:       consts.BalanceRemarkInit,
		})
	})
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			// 并发注册兜底：唯一索引冲突映射为 409，而非 500
			return nil, result.Conflict("账号已被注册", nil)
		}
		return nil, result.System("注册失败", err)
	}
	// 事务提交后清除查重阶段可能写入的 not-found 占位缓存（TTL 1 分钟）：
	// TxInsert 不经过 ExecCtx，占位缓存不会被自动删除，会导致注册后立即登录 401。
	// 清理失败不回滚注册，占位缓存最多 1 分钟自愈，仅记录日志。
	if err := l.svcCtx.UserModel.DeleteCache(l.ctx, newUser); err != nil {
		l.Logger.Errorf("注册后清除用户缓存失败 userId=%d: %v", userId, err)
	}

	tokens, err := l.svcCtx.Tokens.IssueTokens(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	return &types.LoginResp{
		UserId:       userId,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		WsServerUri:  tokens.WsServerUri,
	}, nil
}

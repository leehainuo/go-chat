// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"

	"go-chat/service/user/api/internal/config"
	"go-chat/service/user/api/internal/consts"
	"go-chat/service/user/api/internal/middleware"
	"go-chat/service/user/api/internal/service"
	"go-chat/service/user/api/internal/wslocator"
	balancelogmodel "go-chat/service/user/model/balance_log"
	usermodel "go-chat/service/user/model/user"
	userbalancemodel "go-chat/service/user/model/user_balance"
)

type ServiceContext struct {
	Config            config.Config
	Conn              sqlx.SqlConn
	Redis             *redis.Redis
	UserModel         usermodel.UserModel
	UserBalanceModel  userbalancemodel.UserBalanceModel
	BalanceLogModel   balancelogmodel.BalanceLogModel
	Locator           *wslocator.Locator
	Minio             *minio.Client
	Tokens            *service.TokenService
	JwtAuthMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.MySQL.DataSource)
	rds := redis.MustNewRedis(c.Redis)
	locator := wslocator.NewLocator(c.Etcd.Endpoints, c.Etcd.Prefix)
	mc := newMinioClient(c.Minio)
	ensureBucket(mc, c.Minio.Bucket)

	return &ServiceContext{
		Config:            c,
		Conn:              conn,
		Redis:             rds,
		UserModel:         usermodel.NewUserModel(conn, c.CacheRedis),
		UserBalanceModel:  userbalancemodel.NewUserBalanceModel(conn, c.CacheRedis),
		BalanceLogModel:   balancelogmodel.NewBalanceLogModel(conn, c.CacheRedis),
		Locator:           locator,
		Minio:             mc,
		Tokens:            service.NewTokenService(rds, c.JwtAuth.AccessSecret, locator),
		JwtAuthMiddleware: middleware.NewJwtAuthMiddleware(c.JwtAuth.AccessSecret, rds, consts.AccessTokenPrefix).Handle,
	}
}

func newMinioClient(c config.MinioConf) *minio.Client {
	cli, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: c.Secure,
	})
	if err != nil {
		// 启动容错：MinIO 不可用仅 /uploadUrl 不可用，不阻断服务启动
		logx.Errorf("[minio] 客户端初始化失败: %v（/uploadUrl 接口将不可用）", err)
		return nil
	}
	return cli
}

// ensureBucket 预创建对象桶（设计卡 §5：MinIO 桶 gochat 需预创建），失败仅记录不阻断启动。
func ensureBucket(cli *minio.Client, bucket string) {
	if cli == nil {
		return
	}
	ctx := context.Background()
	exists, err := cli.BucketExists(ctx, bucket)
	if err != nil {
		logx.Errorf("[minio] 检查桶 %s 失败: %v（请确认 MinIO 已启动）", bucket, err)
		return
	}
	if exists {
		return
	}
	if err := cli.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		logx.Errorf("[minio] 创建桶 %s 失败: %v", bucket, err)
		return
	}
	logx.Infof("[minio] 已创建桶 %s", bucket)
}

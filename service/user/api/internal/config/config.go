// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	MySQL struct {
		DataSource string
	}
	Redis redis.RedisConf
	// CacheRedis 为 goctl model 生成的缓存层（sqlc.CachedConn）使用，可与 Redis 同实例
	CacheRedis cache.CacheConf `json:",optional"`
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64 `json:",default=1800"`
	}
	Minio MinioConf
	Etcd struct {
		Endpoints []string `json:",optional"`
		Prefix    string   `json:",default=/services/realtimeservice"`
	}
	// Kafka 通知（DR-5 决策：事务提交后发送，失败仅记日志不回滚；
	// Brokers 为空时降级为仅日志，不阻断任何业务流程）
	Kafka struct {
		Brokers []string `json:",optional"`
	}
}

type MinioConf struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string `json:",default=gochat"`
	Secure    bool   `json:",default=false"`
}

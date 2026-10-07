package config

import (
	"github.com/IM_System/pkg/serviceauth"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ReadQueryAuth serviceauth.Config `json:",optional"`

	Mysql struct {
		DataSource string
	}

	Cache cache.CacheConf

	Redisx redis.RedisConf

	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/IM_System/apps/social/api/internal/config"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/IM_System/pkg/interceptor"
	"github.com/IM_System/pkg/middleware"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var retryPolicy = `{
  "methodConfig": [{
	"name": [{"service": "social.Social"}],
	"waitForReady": true,
	"retryPolicy": {
	  "MaxAttempts": 5,
	  "InitialBackoff": "0.001s",
	  "MaxBackoff": "0.002s",
	  "BackoffMultiplier": 1.0,
	  "RetryableStatusCodes": ["UNKNOWN","DEADLINE_EXCEEDED"]
	}
  }]
}`

type ServiceContext struct {
	Config config.Config

	IdempotenceMiddleware rest.Middleware
	LimitMiddleware       rest.Middleware
	*redis.Redis

	UserRpc   userclient.User
	SocialRpc socialclient.Social
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,

		IdempotenceMiddleware: middleware.NewIdempotenceMiddleware().Handler,
		LimitMiddleware:       middleware.NewLimitMiddleware(c.Redisx).TokenLimitHandler(1, 100),
		Redis:                 redis.MustNewRedis(c.Redisx),

		UserRpc: userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		SocialRpc: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
			zrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy)),
			zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient))),
	}
}

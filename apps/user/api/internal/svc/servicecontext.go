// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/IM_System/apps/user/api/internal/config"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/IM_System/pkg/middleware"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var retryPolicy = `{
  "methodConfig": [{
    "name": [{"service": "user.User"}],
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

	userclient.User
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,

		IdempotenceMiddleware: middleware.NewIdempotenceMiddleware().Handler,
		LimitMiddleware:       middleware.NewLimitMiddleware(c.Redisx).TokenLimitHandler(1, 100),
		Redis:                 redis.MustNewRedis(c.Redisx),

		User: userclient.NewUser(zrpc.MustNewClient(c.UserRpc, zrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy)))),
	}
}

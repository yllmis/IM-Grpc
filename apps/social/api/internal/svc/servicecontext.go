// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/IM_System/apps/social/api/internal/config"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	*redis.Redis

	UserRpc   userclient.User
	SocialRpc socialclient.Social
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,

		Redis: redis.MustNewRedis(c.Redisx),

		UserRpc:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		SocialRpc: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
	}
}

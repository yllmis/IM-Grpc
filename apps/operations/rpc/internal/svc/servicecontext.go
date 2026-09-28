package svc

import (
	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/operations/rpc/internal/config"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// 全部只读依赖
	ChatLogModel immodels.ChatLogModel
	EventModel   operationsmodels.EventModel

	UserRpc   userclient.User
	SocialRpc socialclient.Social
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		ChatLogModel: immodels.MustChatLogModel(c.Mongo.Url, c.Mongo.Db),
		EventModel:   operationsmodels.MustEventModel(c.Mongo.Url, c.Mongo.Db),
		UserRpc:      userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		SocialRpc:    socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
	}
}

// MaxLimit 返回查询上限。
func (s *ServiceContext) MaxLimit() int32 {
	if s.Config.Query.MaxLimit > 0 {
		return s.Config.Query.MaxLimit
	}
	return 200
}

// DefaultLimit 返回默认分页大小。
func (s *ServiceContext) DefaultLimit() int32 {
	if s.Config.Query.DefaultLimit > 0 {
		return s.Config.Query.DefaultLimit
	}
	return 50
}

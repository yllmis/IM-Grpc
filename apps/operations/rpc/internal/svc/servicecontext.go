package svc

import (
	"github.com/IM_System/apps/operations/rpc/internal/config"
	"github.com/IM_System/apps/operations/rpc/internal/faultinject"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
)

type ServiceContext struct {
	Config config.Config

	// 全部只读依赖
	EventModel operationsmodels.EventModel
	// FaultInjection 只改变匹配测试 ID 的查询响应，不触碰 IM 主链路。
	FaultInjection faultinject.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	s := &ServiceContext{
		Config:         c,
		EventModel:     operationsmodels.MustEventModel(c.Mongo.Url, c.Mongo.Db),
		FaultInjection: c.FaultInjection,
	}
	return s
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

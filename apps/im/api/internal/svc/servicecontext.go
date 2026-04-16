// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/IM_System/apps/im/api/internal/config"
	"github.com/IM_System/apps/im/rpc/imclient"
)

type ServiceContext struct {
	Config config.Config

	imclient.Im
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}

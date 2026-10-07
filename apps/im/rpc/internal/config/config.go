package config

import (
	"github.com/IM_System/pkg/serviceauth"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ReadQueryAuth serviceauth.Config `json:",optional"`

	Mongo struct {
		Url string
		Db  string
	}
}

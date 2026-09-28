package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf

	ListenOn string

	MsgChatTransfer kq.KqConf
	MsgReadTransfer kq.KqConf

	Redisx redis.RedisConf

	Mongo struct {
		Url string
		Db  string
	}

	SocialRpc zrpc.RpcClientConf

	Ws struct {
		Host string
	}

	MsgReadHandler struct {
		GroupMsgReadHandler          int
		GroupMsgReadRecordDelayTime  int64
		GroupMsgReadRecordDelayCount int
	}

	// DeliveryObservation 观测开关，默认关闭。第一版修改需重启，不做热更新。
	DeliveryObservation struct {
		Enabled       bool
		PersistEvents bool
		AckMode       string
		InstanceId    string
		BufferSize    int
	}
}

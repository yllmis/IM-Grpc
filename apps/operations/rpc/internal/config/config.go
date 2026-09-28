package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mongo struct {
		Url string
		Db  string
	}

	// 只读下游
	UserRpc   zrpc.RpcClientConf
	SocialRpc zrpc.RpcClientConf

	// ServiceAuth 服务身份认证（metadata: x-im-service-token）
	ServiceAuth struct {
		Enable bool
		Token  string
		// ExtraTokens 轮换重叠窗口内的旧 token
		ExtraTokens []string
	}

	// DeliveryObservation 只读反映部署观测配置，用于 GetCapabilities 诚实声明
	DeliveryObservation struct {
		Enabled bool
		AckMode string
	}

	// Query 预算
	Query struct {
		// MaxLimit 单次返回上限，超过则 INVALID_ARGUMENT
		MaxLimit int32
		// DefaultLimit limit<=0 时使用
		DefaultLimit int32
	}
}

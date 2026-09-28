package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf

	ListenOn string

	JwtAuth struct {
		AccessSecret string
	}

	Mongo struct {
		Url string
		Db  string
	}

	MsgChatTransfer struct {
		Topic string
		Addrs []string
	}

	MsgReadTransfer struct {
		Topic string
		Addrs []string
	}

	Redisx redis.RedisConf

	// AckMode 是 WS 传输层 ACK 开关（NoAck | OnlyAck | RigorAck）。
	// 与 DeliveryObservation.AckMode 必须一致，否则拒绝启动。
	// 默认 NoAck，不改变现有行为。
	AckMode string

	// MessageIdentity 控制稳定消息 ID 的生成与回显。
	// 默认 false：不生成、不进 Kafka、回包不带 serverMessageId（零行为变化）。
	// 切换 true 前必须先部署兼容 messageId 的 task-mq（见 docs/compatibility-and-rollout.md 闸门 A）。
	MessageIdentity struct {
		EmitServerMessageId bool
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

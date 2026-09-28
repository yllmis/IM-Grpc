package svc

import (
	"context"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/internal/config"
	"github.com/IM_System/apps/task/mq/mqclient"
	"github.com/IM_System/pkg/observation"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config config.Config

	*redis.Redis

	immodels.ChatLogModel

	mqclient.MsgChatTransferClient
	mqclient.MsgReadTransferClient

	// ObservationSink 旁路观测；默认 Noop，失败不影响业务
	ObservationSink observation.ObservationSink
}

func NewServiceContext(c config.Config) *ServiceContext {
	sink := newObservationSink(c)

	chatClient := mqclient.WithObservationSink(
		mqclient.NewMsgChatTransferClient(c.MsgChatTransfer.Addrs, c.MsgChatTransfer.Topic),
		sink,
	)

	return &ServiceContext{
		Config:                c,
		Redis:                 redis.MustNewRedis(c.Redisx),
		MsgChatTransferClient: chatClient,
		ChatLogModel:          immodels.MustChatLogModel(c.Mongo.Url, c.Mongo.Db),
		MsgReadTransferClient: mqclient.NewMsgReadTransferClient(c.MsgReadTransfer.Addrs, c.MsgReadTransfer.Topic),
		ObservationSink:       sink,
	}
}

// newObservationSink 按配置构建 Sink。默认 DeliveryObservation.Enabled=false → Noop。
func newObservationSink(c config.Config) observation.ObservationSink {
	cfg := observation.DeliveryObservation{
		Enabled:       c.DeliveryObservation.Enabled,
		PersistEvents: c.DeliveryObservation.PersistEvents,
		AckMode:       c.DeliveryObservation.AckMode,
		InstanceId:    c.DeliveryObservation.InstanceId,
		BufferSize:    c.DeliveryObservation.BufferSize,
	}
	if !cfg.Enabled {
		return observation.Nop()
	}
	if !cfg.PersistEvents {
		cfg.PersistEvents = true
	}

	sink, _, err := observation.NewSinkWithSource(
		context.Background(),
		cfg,
		c.Mongo.Url,
		c.Mongo.Db,
		observation.SourceImWs,
	)
	if err != nil {
		// 观测初始化失败不得阻止业务服务启动
		return observation.Nop()
	}
	return sink
}

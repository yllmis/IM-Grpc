package handler

import (
	"time"

	"github.com/IM_System/apps/task/mq/internal/consumer"
	msgtransfer "github.com/IM_System/apps/task/mq/internal/handler/msgTransfer"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/internal/telemetry"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type Listen struct {
	svc *svc.ServiceContext
}

func NewListen(svc *svc.ServiceContext) *Listen {
	return &Listen{
		svc: svc,
	}
}

func (l *Listen) Services() []service.Service {
	chatConsumer, err := consumer.NewChatConsumer(
		l.svc.Config.MsgChatTransfer,
		l.svc.Config.MessageRetry,
		l.svc.Config.ConsumerScaling.ExpectedPartitions,
		msgtransfer.NewMsgChatTransfer(l.svc),
		l.svc.DeadLetter,
		l.svc.Metrics,
	)
	if err != nil {
		panic(err)
	}
	services := []service.Service{chatConsumer}
	if l.svc.Config.KafkaMonitoring.Enabled {
		interval := time.Duration(l.svc.Config.KafkaMonitoring.LagIntervalSec) * time.Second
		monitor, err := telemetry.NewService(l.svc.Config.MsgChatTransfer,
			l.svc.Config.KafkaMonitoring.ListenOn, interval, l.svc.Metrics)
		if err != nil {
			panic(err)
		}
		services = append(services, monitor)
	}
	services = append(services,
		kq.MustNewQueue(l.svc.Config.MsgReadTransfer, msgtransfer.NewMsgReadTransfer(l.svc)),
	)
	return services
}

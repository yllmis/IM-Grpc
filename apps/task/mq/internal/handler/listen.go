package handler

import (
	msgtransfer "github.com/IM_System/apps/task/mq/internal/handler/msgTransfer"
	"github.com/IM_System/apps/task/mq/internal/svc"
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
	return []service.Service{
		kq.MustNewQueue(l.svc.Config.MsgChatTransfer, msgtransfer.NewMsgChatTransfer(l.svc)),
		kq.MustNewQueue(l.svc.Config.MsgReadTransfer, msgtransfer.NewMsgReadTransfer(l.svc)),
	}
}

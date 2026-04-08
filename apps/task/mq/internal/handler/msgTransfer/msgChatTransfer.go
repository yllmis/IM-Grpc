package msgtransfer

import (
	"context"
	"fmt"

	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type MsgChatTransfer struct {
	logx.Logger
	svc *svc.ServiceContext
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{
		Logger: logx.WithContext(context.Background()),
		svc:    svc,
	}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error { // 消费者，有更新消息时会调用这个方法
	fmt.Printf("consume msg, key: %s, value: %s\n", key, value)
	return nil
}

package mqclient

import (
	"context"
	"encoding/json"

	"github.com/IM_System/apps/task/mq/mq"
	"github.com/zeromicro/go-queue/kq"
)

type MsgChatTransferClient interface {
	Push(msg *mq.MsgChatTransfer) error
}

type msgChatTransferClient struct {
	push *kq.Pusher
}

// NewMsgChatTransferClient(地址，kafka topic，其它参数配置项)
func NewMsgChatTransferClient(addr []string, topic string, opts ...kq.PushOption) MsgChatTransferClient {
	return &msgChatTransferClient{
		push: kq.NewPusher(addr, topic, opts...),
	}
}

func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
	body, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	return c.push.Push(context.Background(), string(body))
}

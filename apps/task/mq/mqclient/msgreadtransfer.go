package mqclient

import (
	"context"
	"encoding/json"

	"github.com/IM_System/apps/task/mq/mq"
	"github.com/zeromicro/go-queue/kq"
)

type MsgReadTransferClient interface {
	Push(msg *mq.MsgMarkRead) error
}

type msgReadTransferClient struct {
	push *kq.Pusher
}

// NewMsgReadTransferClient(地址，kafka topic，其它参数配置项)
func NewMsgReadTransferClient(addr []string, topic string, opts ...kq.PushOption) MsgReadTransferClient {
	return &msgReadTransferClient{
		push: kq.NewPusher(addr, topic, opts...),
	}
}

func (c *msgReadTransferClient) Push(msg *mq.MsgMarkRead) error {
	body, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	return c.push.Push(context.Background(), string(body))
}

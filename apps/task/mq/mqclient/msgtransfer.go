package mqclient

import (
	"context"
	"encoding/json"

	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/observation"
	"github.com/zeromicro/go-queue/kq"
)

type MsgChatTransferClient interface {
	Push(msg *mq.MsgChatTransfer) error
}

type msgChatTransferClient struct {
	push *kq.Pusher
	sink observation.ObservationSink
}

// NewMsgChatTransferClient(地址，kafka topic，其它参数配置项)
func NewMsgChatTransferClient(addr []string, topic string, opts ...kq.PushOption) MsgChatTransferClient {
	return &msgChatTransferClient{
		push: kq.NewPusher(addr, topic, opts...),
		sink: observation.Nop(),
	}
}

// WithObservationSink 注入观测 Sink（默认 Noop）。观测失败不影响 Push 返回值。
func WithObservationSink(c MsgChatTransferClient, sink observation.ObservationSink) MsgChatTransferClient {
	cc, ok := c.(*msgChatTransferClient)
	if !ok {
		return c
	}
	if sink == nil {
		sink = observation.Nop()
	}
	cc.sink = sink
	return cc
}

func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
	body, err := json.Marshal(msg)

	if err != nil {
		c.observe(msg, false)
		return err
	}

	err = c.push.Push(context.Background(), string(body))
	c.observe(msg, err == nil)
	return err
}

// observe 旁路记录 kafka_published / kafka_publish_failed，不改变 Push 语义。
func (c *msgChatTransferClient) observe(msg *mq.MsgChatTransfer, ok bool) {
	if msg == nil {
		return
	}
	// 无稳定 messageId 时不写消息事件（契约要求 messageId 必填）
	if msg.MessageId == "" {
		return
	}

	b := observation.MessageEventBuilder{
		MessageID:       msg.MessageId,
		ClientMessageID: msg.ClientMessageId,
		CorrelationID:   msg.CorrelationId,
		ConversationID:  msg.ConversationId,
		SenderID:        msg.SendId,
		ReceiverID:      msg.RecvId,
		Source:          observation.SourceMqPush,
	}

	ctx := context.Background()
	if ok {
		observation.SafeRecord(ctx, c.sink, b.KafkaPublished())
	} else {
		observation.SafeRecord(ctx, c.sink, b.KafkaPublishFailed(observation.ErrCodeKafkaPublishFailed))
	}
}

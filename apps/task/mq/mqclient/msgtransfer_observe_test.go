package mqclient

import (
	"context"
	"sync"
	"testing"

	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/observation"
)

type recSink struct {
	mu     sync.Mutex
	events []observation.MessageEvent
}

func (r *recSink) Record(_ context.Context, ev observation.MessageEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
	return nil
}

func (r *recSink) all() []observation.MessageEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]observation.MessageEvent, len(r.events))
	copy(out, r.events)
	return out
}

func TestObserve_RequiresMessageId(t *testing.T) {
	sink := &recSink{}
	c := &msgChatTransferClient{sink: sink}

	// 无稳定 messageId：不写消息事件
	c.observe(&mq.MsgChatTransfer{SendId: "u1"}, true)
	if len(sink.all()) != 0 {
		t.Fatalf("must skip events without messageId, got %d", len(sink.all()))
	}
}

func TestObserve_KafkaPublished(t *testing.T) {
	sink := &recSink{}
	c := &msgChatTransferClient{sink: sink}

	msg := &mq.MsgChatTransfer{
		MessageId:       observation.NewEventID(),
		ClientMessageId: "cli-1",
		ConversationId:  "c1",
		SendId:          "u1",
		RecvId:          "u2",
		ChatType:        constants.SingleChatType,
	}
	c.observe(msg, true)

	events := sink.all()
	if len(events) != 1 || events[0].EventType != observation.EventKafkaPublished {
		t.Fatalf("want kafka_published, got %+v", events)
	}
	if events[0].MessageID != msg.MessageId || events[0].Source != observation.SourceMqPush {
		t.Fatalf("event fields wrong: %+v", events[0])
	}
}

func TestObserve_KafkaPublishFailed(t *testing.T) {
	sink := &recSink{}
	c := &msgChatTransferClient{sink: sink}

	msg := &mq.MsgChatTransfer{MessageId: observation.NewEventID(), SendId: "u1"}
	c.observe(msg, false)

	events := sink.all()
	if len(events) != 1 || events[0].EventType != observation.EventKafkaPublishFailed {
		t.Fatalf("want kafka_publish_failed, got %+v", events)
	}
	if events[0].ErrorCode != observation.ErrCodeKafkaPublishFailed {
		t.Fatalf("errorCode=%s", events[0].ErrorCode)
	}
}

func TestWithObservationSink_DefaultNoop(t *testing.T) {
	c := NewMsgChatTransferClient([]string{"127.0.0.1:9092"}, "topic")
	cc := WithObservationSink(c, nil).(*msgChatTransferClient)
	if cc.sink == nil {
		t.Fatal("sink must default to Noop, not nil")
	}
	// Noop 不 panic
	cc.observe(&mq.MsgChatTransfer{}, true)
}

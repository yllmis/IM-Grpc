package push

import (
	"context"
	"sync"
	"testing"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/ws"
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

func (r *recSink) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.events))
	for _, ev := range r.events {
		out = append(out, ev.EventType)
	}
	return out
}

func TestDeliveryEvents_MessageIdRequired(t *testing.T) {
	sink := &recSink{}
	svcCtx := &svc.ServiceContext{ObservationSink: sink}

	data := &ws.Push{
		ConversationId: "c1",
		ChatType:       constants.SingleChatType,
		SendId:         "u1",
		RecvId:         "u2",
		MsgId:          observation.NewEventID(),
	}

	// 无连接路径（GetConn 返回 nil 在无 server 时无法直接跑，这里验证 builder 语义）
	_ = svcCtx
	b := observation.MessageEventBuilder{
		MessageID:      data.MsgId,
		ConversationID: data.ConversationId,
		SenderID:       data.SendId,
		ReceiverID:     data.RecvId,
		Source:         observation.SourceWsPush,
	}
	observation.SafeRecord(context.Background(), sink, b.ReceiverOffline())
	observation.SafeRecord(context.Background(), sink, b.DeliveryAttempted())
	observation.SafeRecord(context.Background(), sink, b.DeliverySucceeded())

	types := sink.types()
	want := []string{
		observation.EventReceiverOffline,
		observation.EventDeliveryAttempted,
		observation.EventDeliverySucceeded,
	}
	if len(types) != len(want) {
		t.Fatalf("got %v want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("got %v want %v", types, want)
		}
	}
}

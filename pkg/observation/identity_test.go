package observation

import (
	"context"
	"sync"
	"testing"
)

// recordingSink 收集事件，模拟 message_events 幂等语义。
type recordingSink struct {
	mu    sync.Mutex
	byID  map[string]MessageEvent
	order []string
}

func newRecordingSink() *recordingSink {
	return &recordingSink{byID: make(map[string]MessageEvent)}
}

func (r *recordingSink) Record(_ context.Context, ev MessageEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[ev.EventID]; ok {
		return nil // 幂等：同 eventId 不重复
	}
	r.byID[ev.EventID] = ev
	r.order = append(r.order, ev.EventID)
	return nil
}

func (r *recordingSink) events() []MessageEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]MessageEvent, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.byID[id])
	}
	return out
}

// TestMessageIDChain_贯通 模拟 accepted → kafka → persisted 使用同一 messageId。
func TestMessageIDChain_贯通(t *testing.T) {
	sink := newRecordingSink()
	msgId := NewEventID()
	clientId := "client-req-1"

	b := MessageEventBuilder{
		MessageID:       msgId,
		ClientMessageID: clientId,
		ConversationID:  "c1",
		SenderID:        "u1",
		ReceiverID:      "u2",
		Source:          SourceImWs,
	}

	SafeRecord(context.Background(), sink, b.Accepted())
	SafeRecord(context.Background(), sink, b.KafkaPublished())

	// 消费侧：同 messageId，source 切 task-mq
	consumed := b
	consumed.Source = SourceTaskMq
	SafeRecord(context.Background(), sink, consumed.KafkaConsumed())
	SafeRecord(context.Background(), sink, consumed.Persisted())

	events := sink.events()
	if len(events) != 4 {
		t.Fatalf("want 4 events, got %d", len(events))
	}
	for _, ev := range events {
		if ev.MessageID != msgId {
			t.Fatalf("messageId must be stable across chain, got %s want %s", ev.MessageID, msgId)
		}
		if ev.ClientMessageID != clientId {
			t.Fatalf("clientMessageId must propagate, got %s", ev.ClientMessageID)
		}
		if ev.CorrelationID != msgId {
			t.Fatalf("correlationId must default to messageId")
		}
	}

	wantTypes := []string{EventAccepted, EventKafkaPublished, EventKafkaConsumed, EventPersisted}
	for i, ev := range events {
		if ev.EventType != wantTypes[i] {
			t.Fatalf("order[%d]=%s want %s", i, ev.EventType, wantTypes[i])
		}
	}
}

// TestLegacyMessage_Compat 旧消息无 messageId，消费端生成并标注 compat-legacy。
func TestLegacyMessage_Compat(t *testing.T) {
	sink := newRecordingSink()
	// 模拟 ResolveMessageId legacy 路径
	legacyId := NewEventID()
	b := MessageEventBuilder{
		MessageID: legacyId,
		Source:    SourceCompatLegacy,
		SenderID:  "u1",
	}
	SafeRecord(context.Background(), sink, b.KafkaConsumed())
	SafeRecord(context.Background(), sink, b.Persisted())

	for _, ev := range sink.events() {
		if ev.Source != SourceCompatLegacy {
			t.Fatalf("legacy source required, got %s", ev.Source)
		}
		if ev.MessageID != legacyId {
			t.Fatalf("legacy must still carry resolved id")
		}
		if ev.ClientMessageID != "" {
			t.Fatal("legacy must not invent clientMessageId")
		}
	}
}

// TestEventIdempotency 同 eventId 重复写不产生副本。
func TestEventIdempotency(t *testing.T) {
	sink := newRecordingSink()
	ev := MessageEventBuilder{MessageID: NewEventID(), Source: SourceImWs}.Accepted()
	SafeRecord(context.Background(), sink, ev)
	SafeRecord(context.Background(), sink, ev) // duplicate

	if got := len(sink.events()); got != 1 {
		t.Fatalf("duplicate eventId must not duplicate record, got %d", got)
	}
}

// TestDeliveryEvents_PerReceiver 群聊每个接收方独立事件。
func TestDeliveryEvents_PerReceiver(t *testing.T) {
	sink := newRecordingSink()
	msgId := NewEventID()
	receivers := []string{"u2", "u3", "u4"}

	for _, r := range receivers {
		b := MessageEventBuilder{
			MessageID:      msgId,
			ConversationID: "group-1",
			SenderID:       "u1",
			ReceiverID:     r,
			Source:         SourceWsPush,
		}
		SafeRecord(context.Background(), sink, b.DeliveryAttempted())
		SafeRecord(context.Background(), sink, b.DeliverySucceeded())
	}

	byRecv := map[string]int{}
	for _, ev := range sink.events() {
		byRecv[ev.ReceiverID]++
		if ev.MessageID != msgId {
			t.Fatalf("messageId mismatch")
		}
	}
	if len(byRecv) != 3 {
		t.Fatalf("each receiver must have independent records, got %v", byRecv)
	}
	for r, n := range byRecv {
		if n != 2 {
			t.Fatalf("receiver %s want 2 events got %d", r, n)
		}
	}
}

// TestNoFakeAck 默认/OnlyAck 不产生 ack_*。
func TestNoFakeAck(t *testing.T) {
	sink := newRecordingSink()
	b := MessageEventBuilder{MessageID: NewEventID(), Source: SourceImWs}

	// 只写业务侧真实发生的 read_confirmed，不写 ack_*
	SafeRecord(context.Background(), sink, b.ReadConfirmed())

	for _, ev := range sink.events() {
		if ev.EventType == EventAckReceived || ev.EventType == EventAckTimeout {
			t.Fatalf("must not fabricate ack events: %s", ev.EventType)
		}
	}
}

// TestRedaction 事件构建器不得携带正文。
func TestRedaction(t *testing.T) {
	b := MessageEventBuilder{
		MessageID:      NewEventID(),
		ConversationID: "c1",
		SenderID:       "u1",
		Source:         SourceTaskMq,
	}
	ev := b.Persisted()
	if ev.Metadata != nil {
		for k := range ev.Metadata {
			switch k {
			case "connectionId", "instanceId", "reason", "droppedCount", "gapStartAt", "gapEndAt", "transport", "ackSeq":
			default:
				// 不允许任意键值带正文
				if len(ev.Metadata[k]) > 128 {
					t.Fatalf("metadata too large: %s", k)
				}
			}
		}
	}
	// MessageEvent 结构本身无 Content 字段
	if ev.Validate() != nil {
		t.Fatal("event must be valid")
	}
}

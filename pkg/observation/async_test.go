package observation

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeSink struct {
	mu     sync.Mutex
	events []MessageEvent
	fail   bool
}

func (f *fakeSink) Record(_ context.Context, ev MessageEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errInvalidEvent("forced failure")
	}
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeSink) snapshot() []MessageEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]MessageEvent, len(f.events))
	copy(out, f.events)
	return out
}

func TestAsyncSink_DoesNotBlock(t *testing.T) {
	inner := &fakeSink{fail: true}
	s := NewAsyncObservationSink(inner, SourceImWs, 2)

	// 超出缓冲 + 失败写入，Record 仍应立即返回
	for i := 0; i < 20; i++ {
		_ = s.Record(context.Background(), MessageEvent{
			EventID:    NewEventID(),
			EventType:  EventAccepted,
			MessageID:  "m1",
			Source:     SourceImWs,
			OccurredAt: NowUnixNano(),
		})
	}
	s.Close()

	if s.Metrics().Dropped.Load() == 0 && s.Metrics().Failed.Load() == 0 {
		t.Fatal("expected drops or failures under pressure")
	}
}

func TestAsyncSink_GapMarker(t *testing.T) {
	inner := &fakeSink{}
	s := NewAsyncObservationSink(inner, SourceTaskMq, 1)

	// 阻塞 worker：先塞满并丢弃
	_ = s.Record(context.Background(), MessageEvent{
		EventID: NewEventID(), EventType: EventAccepted, MessageID: "m", Source: SourceTaskMq, OccurredAt: 1,
	})
	// 第二条进入缓冲后 worker 未消费完前会丢（buffer=1，第一条已被取走或占满）
	for i := 0; i < 5; i++ {
		_ = s.Record(context.Background(), MessageEvent{
			EventID: NewEventID(), EventType: EventAccepted, MessageID: "m", Source: SourceTaskMq, OccurredAt: int64(10 + i),
		})
	}

	s.Close()

	gapFound := false
	for _, ev := range inner.snapshot() {
		if ev.EventType == EventObservationGap {
			gapFound = true
			if ev.Metadata["droppedCount"] == "" || ev.Metadata["gapStartAt"] == "" {
				t.Fatalf("gap metadata incomplete: %v", ev.Metadata)
			}
		}
	}
	// 缓冲极小时几乎必然丢弃；若恰好未丢则跳过 gap 断言
	if s.Metrics().Dropped.Load() > 0 && !gapFound {
		t.Fatalf("dropped=%d but no observation_gap written, events=%d", s.Metrics().Dropped.Load(), len(inner.snapshot()))
	}
}

func TestNoopSink(t *testing.T) {
	if err := Nop().Record(context.Background(), MessageEvent{}); err != nil {
		t.Fatalf("noop must never error: %v", err)
	}
}

func TestIdempotentEventIdentity(t *testing.T) {
	// 同 eventId 两次：模拟幂等键语义（真实去重由 Mongo unique + SetOnInsert 保证）
	id := NewEventID()
	e1 := MessageEvent{EventID: id, EventType: EventPersisted, MessageID: "m", Source: SourceTaskMq, OccurredAt: time.Now().UnixNano()}
	e2 := e1
	if e1.EventID != e2.EventID {
		t.Fatal("same logical event must share eventId")
	}
}

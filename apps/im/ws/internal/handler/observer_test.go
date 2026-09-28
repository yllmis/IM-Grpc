package handler

import (
	"context"
	"sync"
	"testing"

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

func TestSinkObserver_ConnectionEvents(t *testing.T) {
	sink := &recSink{}
	obs := NewSinkObserver(sink, "inst-1")

	obs.OnConnect("u1", "conn-1")
	obs.OnDisconnect("u1", "conn-1", "timeout")

	events := sink.all()
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].EventType != observation.EventOnline || events[0].SenderID != "u1" {
		t.Fatalf("online event wrong: %+v", events[0])
	}
	if events[0].Metadata["instanceId"] != "inst-1" || events[0].Metadata["connectionId"] != "conn-1" {
		t.Fatalf("online metadata wrong: %v", events[0].Metadata)
	}
	if events[1].EventType != observation.EventOffline || events[1].Metadata["reason"] != "timeout" {
		t.Fatalf("offline event wrong: %+v", events[1])
	}
	// 连接事件 messageId 可空
	if err := events[0].Validate(); err != nil {
		t.Fatalf("online event must validate: %v", err)
	}
}

func TestSinkObserver_AckEvents(t *testing.T) {
	sink := &recSink{}
	obs := NewSinkObserver(sink, "inst-1")

	obs.OnAckReceived("u1", "frame-1", 2)
	obs.OnAckTimeout("u1", "frame-1", 1)

	events := sink.all()
	if events[0].EventType != observation.EventAckReceived {
		t.Fatalf("want ack_received, got %s", events[0].EventType)
	}
	if events[1].EventType != observation.EventAckTimeout || events[1].ErrorCode != observation.ErrCodeAckTimeout {
		t.Fatalf("want ack_timeout, got %+v", events[1])
	}
	if events[0].Metadata["transport"] != "ws-ack" {
		t.Fatalf("metadata=%v", events[0].Metadata)
	}
	// 脱敏：不得携带正文/Token/原始 payload
	for _, ev := range events {
		for k, v := range ev.Metadata {
			if len(v) > 128 {
				t.Fatalf("metadata value too long: %s=%s", k, v)
			}
		}
	}
}

func TestSinkObserver_NilSinkSafe(t *testing.T) {
	obs := NewSinkObserver(nil, "")
	obs.OnConnect("u1", "c1")
	obs.OnDisconnect("u1", "c1", "disconnect")
	obs.OnAckReceived("u1", "f1", 1)
	obs.OnAckTimeout("u1", "f1", 1)
}

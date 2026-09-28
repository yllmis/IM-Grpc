package observation

import "testing"

func TestMessageEventValidate(t *testing.T) {
	valid := MessageEvent{
		EventID:    NewEventID(),
		EventType:  EventAccepted,
		MessageID:  NewEventID(),
		Source:     SourceImWs,
		OccurredAt: NowUnixNano(),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	cases := []struct {
		name  string
		event MessageEvent
	}{
		{"missing eventId", MessageEvent{EventType: EventAccepted, MessageID: "m", Source: SourceImWs, OccurredAt: 1}},
		{"missing eventType", MessageEvent{EventID: "e", MessageID: "m", Source: SourceImWs, OccurredAt: 1}},
		{"missing source", MessageEvent{EventID: "e", EventType: EventAccepted, MessageID: "m", OccurredAt: 1}},
		{"bad occurredAt", MessageEvent{EventID: "e", EventType: EventAccepted, MessageID: "m", Source: SourceImWs, OccurredAt: 0}},
		{"missing messageId", MessageEvent{EventID: "e", EventType: EventPersisted, Source: SourceTaskMq, OccurredAt: 1}},
	}
	for _, tc := range cases {
		if err := tc.event.Validate(); err == nil {
			t.Fatalf("%s: expected validation error", tc.name)
		}
	}

	// 连接事件与 observation_gap 允许无 messageId
	for _, et := range []string{EventOnline, EventOffline, EventObservationGap} {
		ev := MessageEvent{EventID: NewEventID(), EventType: et, Source: SourceImWs, OccurredAt: NowUnixNano()}
		if err := ev.Validate(); err != nil {
			t.Fatalf("%s must allow empty messageId: %v", et, err)
		}
	}
}

func TestNewEventID(t *testing.T) {
	a, b := NewEventID(), NewEventID()
	if a == "" || b == "" || a == b {
		t.Fatalf("event ids must be unique non-empty, got %q %q", a, b)
	}
	if _, ok := ParseObjectIDHex(a); !ok {
		t.Fatalf("eventId must be 24hex ObjectID, got %q", a)
	}
}

func TestNewObservationGap(t *testing.T) {
	gap := NewObservationGap(SourceImWs, 5, 100, 200, "buffer_full")
	if gap.EventType != EventObservationGap {
		t.Fatalf("type = %s", gap.EventType)
	}
	if gap.Metadata["droppedCount"] != "5" || gap.Metadata["reason"] != "buffer_full" {
		t.Fatalf("metadata = %v", gap.Metadata)
	}
	if err := gap.Validate(); err != nil {
		t.Fatalf("gap must be valid: %v", err)
	}
}

func TestDeliveryObservationConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config must be valid: %v", err)
	}
	if cfg.Enabled || cfg.AckMode != AckModeNoAck {
		t.Fatalf("defaults must disable observation: %+v", cfg)
	}

	bad := cfg
	bad.Enabled = true
	bad.PersistEvents = false
	if err := bad.Validate(); err == nil {
		t.Fatal("Enabled without PersistEvents must fail")
	}

	badAck := cfg
	badAck.AckMode = "FakeAck"
	if err := badAck.Validate(); err == nil {
		t.Fatal("invalid AckMode must fail")
	}
}

func TestParseAckMode(t *testing.T) {
	if m, err := ParseAckMode("RigorAck"); err != nil || m != AckModeRigorAck {
		t.Fatalf("got %q %v", m, err)
	}
	if m, err := ParseAckMode(""); err != nil || m != AckModeNoAck {
		t.Fatalf("empty must default NoAck, got %q %v", m, err)
	}
	if _, err := ParseAckMode("nope"); err == nil {
		t.Fatal("invalid mode must error")
	}
}

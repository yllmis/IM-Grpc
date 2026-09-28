package observation

import (
	"context"
	"testing"
)

func TestMessageEventBuilder_Types(t *testing.T) {
	b := MessageEventBuilder{
		MessageID:       NewEventID(),
		ClientMessageID: "cli-1",
		ConversationID:  "c1",
		SenderID:        "u1",
		ReceiverID:      "u2",
		Source:          SourceImWs,
	}

	cases := []struct {
		name string
		ev   MessageEvent
		want string
	}{
		{"accepted", b.Accepted(), EventAccepted},
		{"kafka_published", b.KafkaPublished(), EventKafkaPublished},
		{"kafka_consumed", b.KafkaConsumed(), EventKafkaConsumed},
		{"persisted", b.Persisted(), EventPersisted},
		{"delivery_attempted", b.DeliveryAttempted(), EventDeliveryAttempted},
		{"delivery_succeeded", b.DeliverySucceeded(), EventDeliverySucceeded},
		{"read_confirmed", b.ReadConfirmed(), EventReadConfirmed},
		{"kafka_publish_failed", b.KafkaPublishFailed(ErrCodeKafkaPublishFailed), EventKafkaPublishFailed},
		{"persist_failed", b.PersistFailed(ErrCodePersistFailed), EventPersistFailed},
		{"delivery_failed", b.DeliveryFailed(ErrCodeDeliveryWriteFailed), EventDeliveryFailed},
		{"receiver_offline", b.ReceiverOffline(), EventReceiverOffline},
	}

	for _, tc := range cases {
		if tc.ev.EventType != tc.want {
			t.Fatalf("%s: type=%s want %s", tc.name, tc.ev.EventType, tc.want)
		}
		if tc.ev.MessageID != b.MessageID {
			t.Fatalf("%s: messageId must propagate", tc.name)
		}
		if tc.ev.ClientMessageID != "cli-1" {
			t.Fatalf("%s: clientMessageId must propagate", tc.name)
		}
		if tc.ev.CorrelationID != b.MessageID {
			t.Fatalf("%s: correlationId must default to messageId", tc.name)
		}
		if err := tc.ev.Validate(); err != nil {
			t.Fatalf("%s: invalid event: %v", tc.name, err)
		}
	}

	if b.KafkaPublishFailed(ErrCodeKafkaPublishFailed).ErrorCode != ErrCodeKafkaPublishFailed {
		t.Fatal("failure events must carry errorCode")
	}
	if b.ReceiverOffline().ErrorCode != ErrCodeDeliveryNoConn {
		t.Fatal("receiver_offline must carry DELIVERY_NO_CONN")
	}
}

func TestConnectionEvent(t *testing.T) {
	ev := ConnectionEvent(EventOnline, SourceImWs, "u1", "conn-1", "inst-1", "connect")
	if ev.EventType != EventOnline || ev.SenderID != "u1" {
		t.Fatalf("bad connection event: %+v", ev)
	}
	if ev.Metadata["connectionId"] != "conn-1" || ev.Metadata["instanceId"] != "inst-1" {
		t.Fatalf("metadata missing: %v", ev.Metadata)
	}
	if err := ev.Validate(); err != nil {
		t.Fatalf("connection event must allow empty messageId: %v", err)
	}
}

func TestSafeRecord_NilSink(t *testing.T) {
	// 不应 panic
	SafeRecord(context.Background(), nil, MessageEvent{})
	SafeRecord(context.Background(), Nop(), MessageEvent{})
}

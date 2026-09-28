package mq

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestResolveMessageId_NewFormat(t *testing.T) {
	oid := bson.NewObjectID()
	msg := &MsgChatTransfer{MessageId: oid.Hex()}

	got, legacy := ResolveMessageId(msg)
	if legacy {
		t.Fatalf("expected new-format message, got legacy=true")
	}
	if got != oid {
		t.Fatalf("messageId must be reused, want %s got %s", oid.Hex(), got.Hex())
	}
}

func TestResolveMessageId_LegacyEmpty(t *testing.T) {
	got, legacy := ResolveMessageId(&MsgChatTransfer{})
	if !legacy {
		t.Fatalf("empty messageId must be legacy")
	}
	if got.IsZero() {
		t.Fatalf("legacy path must generate ObjectID")
	}
}

func TestResolveMessageId_LegacyInvalid(t *testing.T) {
	got, legacy := ResolveMessageId(&MsgChatTransfer{MessageId: "not-a-valid-objectid"})
	if !legacy {
		t.Fatalf("invalid messageId must fall back to legacy")
	}
	if got.IsZero() {
		t.Fatalf("legacy fallback must generate ObjectID")
	}
}

func TestResolveMessageId_Nil(t *testing.T) {
	got, legacy := ResolveMessageId(nil)
	if !legacy || got.IsZero() {
		t.Fatalf("nil msg must be legacy with generated id")
	}
}

func TestNormalizeCorrelationId(t *testing.T) {
	oid := bson.NewObjectID().Hex()
	if got := NormalizeCorrelationId(&MsgChatTransfer{}, oid); got != oid {
		t.Fatalf("empty correlationId must default to messageId, got %q", got)
	}
	if got := NormalizeCorrelationId(&MsgChatTransfer{CorrelationId: "corr-1"}, oid); got != "corr-1" {
		t.Fatalf("explicit correlationId must be kept, got %q", got)
	}
}

func TestMsgChatTransfer_JSONBackwardCompatible(t *testing.T) {
	// 旧生产者 payload：无新字段
	oldPayload := `{"conversationId":"c1","chatType":2,"sendId":"u1","recvId":"u2","sendTime":1,"mType":0,"content":"hi"}`
	var msg MsgChatTransfer
	if err := json.Unmarshal([]byte(oldPayload), &msg); err != nil {
		t.Fatalf("old payload must unmarshal: %v", err)
	}
	if msg.MessageId != "" || msg.ClientMessageId != "" || msg.CorrelationId != "" {
		t.Fatalf("old payload must leave new fields empty")
	}

	// 新生产者 payload：含新字段
	newPayload := `{"messageId":"665f1c0000000000000000aa","clientMessageId":"cli-1","correlationId":"corr-1","conversationId":"c1","chatType":2,"sendId":"u1","recvId":"u2","sendTime":1,"mType":0,"content":"hi"}`
	if err := json.Unmarshal([]byte(newPayload), &msg); err != nil {
		t.Fatalf("new payload must unmarshal: %v", err)
	}
	if msg.MessageId != "665f1c0000000000000000aa" || msg.ClientMessageId != "cli-1" || msg.CorrelationId != "corr-1" {
		t.Fatalf("new fields must be parsed: %+v", msg)
	}
}

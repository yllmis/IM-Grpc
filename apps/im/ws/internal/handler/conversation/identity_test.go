package conversation

import "testing"

func TestNewMessageIdentity_EmitOff(t *testing.T) {
	ident := NewMessageIdentity(false, "cli-1")
	if ident.MessageId != "" || ident.ClientMessageId != "" || ident.CorrelationId != "" {
		t.Fatalf("emit=false must not generate ids, got %+v", ident)
	}
}

func TestNewMessageIdentity_EmitOn(t *testing.T) {
	ident := NewMessageIdentity(true, "cli-1")
	if ident.MessageId == "" || len(ident.MessageId) != 24 {
		t.Fatalf("messageId must be 24hex, got %q", ident.MessageId)
	}
	if ident.ClientMessageId != "cli-1" {
		t.Fatalf("clientMessageId must be preserved, got %q", ident.ClientMessageId)
	}
	if ident.CorrelationId != ident.MessageId {
		t.Fatalf("correlationId must default to messageId")
	}

	other := NewMessageIdentity(true, "cli-2")
	if other.MessageId == ident.MessageId {
		t.Fatal("each message must get a fresh messageId")
	}
}

func TestSentResponse_Compat(t *testing.T) {
	// 默认（未开启稳定 ID）：仅 msgId + status，字段名与旧客户端一致
	resp := SentResponse(MessageIdentity{}, "cli-1")
	if resp["msgId"] != "cli-1" {
		t.Fatalf("msgId must be preserved: %v", resp)
	}
	if resp["status"] != "sent" {
		t.Fatalf("status must stay sent: %v", resp)
	}
	if _, ok := resp["serverMessageId"]; ok {
		t.Fatalf("serverMessageId must be absent when disabled: %v", resp)
	}
}

func TestSentResponse_WithServerMessageId(t *testing.T) {
	ident := NewMessageIdentity(true, "cli-1")
	resp := SentResponse(ident, "cli-1")
	if resp["msgId"] != "cli-1" {
		t.Fatalf("old msgId field must remain: %v", resp)
	}
	if resp["status"] != "sent" {
		t.Fatalf("status must remain sent, not delivered: %v", resp)
	}
	if resp["serverMessageId"] != ident.MessageId {
		t.Fatalf("serverMessageId mismatch: %v", resp)
	}
}

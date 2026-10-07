package immodels

import (
	"context"
	"testing"
)

func TestMessageSearchStorageRejectsUnboundedLimit(t *testing.T) {
	model := &messageSearchModel{}
	for _, limit := range []int32{-1, 0, 21} {
		if _, err := model.Search(context.Background(), MessageSearchFilter{Limit: limit}); err == nil {
			t.Fatalf("accepted limit=%d", limit)
		}
	}
}
func TestMessageSearchFilterStaysInMessageDomain(t *testing.T) {
	q := messageSearchQuery(MessageSearchFilter{SenderID: "u1", ReceiverID: "u2", StartTime: 1, EndTime: 100, Limit: 10})
	if q["sendId"] != "u1" || q["recvId"] != "u2" || q["sendTime"] == nil {
		t.Fatalf("%+v", q)
	}
	q = messageSearchQuery(MessageSearchFilter{SenderID: "u1", StartTime: 1, EndTime: 100, Limit: 10})
	if _, ok := q["recvId"]; ok {
		t.Fatal("optional receiver was forced")
	}
}

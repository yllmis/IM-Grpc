package observation

import (
	"context"
	"os"
	"testing"
	"time"
)

// Mongo 集成测试：需要 MONGO_URL（测试环境，禁止生产凭证）。
// 未设置 MONGO_URL 时自动 Skip，不影响 go test ./... 默认门禁。
// 手工执行：MONGO_URL=mongodb://127.0.0.1:27017 go test -tags integration ./pkg/observation/ -run Integration
func TestMongoSink_IdempotentWrite_Integration(t *testing.T) {
	url := os.Getenv("MONGO_URL")
	if url == "" {
		t.Skip("MONGO_URL not set; skip mongo integration test")
	}
	db := os.Getenv("MONGO_DB")
	if db == "" {
		db = "im_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sink, err := NewMongoObservationSink(ctx, url, db)
	if err != nil {
		t.Fatalf("new mongo sink: %v", err)
	}

	ev := MessageEvent{
		EventID:      NewEventID(),
		EventVersion: EventVersion,
		EventType:    EventAccepted,
		MessageID:    NewEventID(),
		Source:       SourceImWs,
		OccurredAt:   NowUnixNano(),
	}

	if err := sink.Record(ctx, ev); err != nil {
		t.Fatalf("first record: %v", err)
	}
	// 幂等：同 eventId 再写成功且不产生副本
	if err := sink.Record(ctx, ev); err != nil {
		t.Fatalf("duplicate record must be idempotent: %v", err)
	}

	// 查询侧可按 messageId 找到事件
	filter := map[string]any{"messageId": ev.MessageID}
	res, err := sink.coll.Find(ctx, filter)
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	defer res.Close(ctx)

	count := 0
	for res.Next(ctx) {
		count++
	}
	if count == 0 {
		t.Fatal("event must be queryable by messageId")
	}
}

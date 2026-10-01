package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/apps/task/mq/internal/dlq"
	"github.com/IM_System/apps/task/mq/internal/telemetry"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type handlerFunc func(context.Context, string, string) error

func (f handlerFunc) Consume(ctx context.Context, key, value string) error { return f(ctx, key, value) }

type fakePublisher struct{ records []dlq.Record }

func (p *fakePublisher) Publish(_ context.Context, _ string, record dlq.Record) error {
	p.records = append(p.records, record)
	return nil
}
func (*fakePublisher) Close() error { return nil }

func TestChatConsumerRetriesAndPublishesDLQ(t *testing.T) {
	attempts := 0
	publisher := &fakePublisher{}
	c, err := NewChatConsumer(kq.KqConf{Brokers: []string{"localhost:9092"}, Topic: "chat", Group: "group"},
		config.MessageRetryConfig{Enabled: true, MaxAttempts: 3, InitialBackoffMs: 1, MaxBackoffMs: 1},
		0,
		handlerFunc(func(context.Context, string, string) error {
			attempts++
			return errors.New("mongo unavailable")
		}), publisher, telemetry.NewMetrics())
	if err != nil {
		t.Fatal(err)
	}
	c.policy.InitialDelay = time.Millisecond
	c.policy.MaxDelay = time.Millisecond
	message := kafka.Message{Key: []byte("key"), Value: []byte("value")}
	handleErr := c.handle(context.Background(), message)
	if handleErr == nil || attempts != 3 {
		t.Fatalf("attempts=%d err=%v", attempts, handleErr)
	}
	if err := c.publishUntilAccepted(context.Background(), message, handleErr); err != nil {
		t.Fatal(err)
	}
	if len(publisher.records) != 1 || publisher.records[0].Attempts != 3 || publisher.records[0].OriginalValue != "value" {
		t.Fatalf("unexpected DLQ records: %#v", publisher.records)
	}
}

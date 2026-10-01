package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IM_System/apps/task/mq/internal/kafkautil"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type Record struct {
	OriginalTopic string `json:"originalTopic"`
	OriginalKey   string `json:"originalKey"`
	OriginalValue string `json:"originalValue"`
	Error         string `json:"error"`
	Attempts      int    `json:"attempts"`
	FailedAt      string `json:"failedAt"`
}

type Publisher interface {
	Publish(ctx context.Context, key string, record Record) error
	Close() error
}

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(conf kq.KqConf, topic string) (*KafkaPublisher, error) {
	transport, err := kafkautil.Transport(conf)
	if err != nil {
		return nil, err
	}
	return &KafkaPublisher{writer: &kafka.Writer{
		Addr:         kafka.TCP(conf.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Transport:    transport,
	}}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, key string, record Record) error {
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: value})
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	err := p.writer.Close()
	if err != nil {
		return fmt.Errorf("close DLQ writer: %w", err)
	}
	return nil
}

func NewRecord(topic, key, value string, cause error, attempts int) Record {
	return Record{OriginalTopic: topic, OriginalKey: key, OriginalValue: value,
		Error: cause.Error(), Attempts: attempts, FailedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

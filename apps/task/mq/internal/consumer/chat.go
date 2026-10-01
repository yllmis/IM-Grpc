package consumer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/apps/task/mq/internal/dlq"
	"github.com/IM_System/apps/task/mq/internal/kafkautil"
	"github.com/IM_System/apps/task/mq/internal/retry"
	"github.com/IM_System/apps/task/mq/internal/telemetry"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChatConsumer struct {
	conf               kq.KqConf
	policy             retry.Policy
	messageRetry       bool
	expectedPartitions int
	dlq                dlq.Publisher
	handler            kq.ConsumeHandler
	metrics            *telemetry.Metrics
	cancel             context.CancelFunc
	readers            []*kafka.Reader
	wg                 sync.WaitGroup
	stopOnce           sync.Once
}

func NewChatConsumer(c kq.KqConf, retryConf config.MessageRetryConfig, expectedPartitions int, handler kq.ConsumeHandler,
	publisher dlq.Publisher, metrics *telemetry.Metrics) (*ChatConsumer, error) {
	if len(c.Brokers) == 0 || c.Topic == "" || c.Group == "" {
		return nil, fmt.Errorf("Kafka chat consumer requires brokers, topic and group")
	}
	return &ChatConsumer{
		conf: c, handler: handler, dlq: publisher, metrics: metrics,
		messageRetry: retryConf.Enabled, expectedPartitions: expectedPartitions,
		policy: retry.Policy{MaxAttempts: retryConf.MaxAttempts,
			InitialDelay: time.Duration(retryConf.InitialBackoffMs) * time.Millisecond,
			MaxDelay:     time.Duration(retryConf.MaxBackoffMs) * time.Millisecond},
	}, nil
}

func (c *ChatConsumer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	if err := c.validatePartitionLayout(ctx); err != nil {
		panic(fmt.Errorf("validate Kafka partition layout: %w", err))
	}
	count := c.conf.Consumers
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		reader, err := newReader(c.conf)
		if err != nil {
			logx.Errorf("create Kafka chat reader: %v", err)
			return
		}
		c.readers = append(c.readers, reader)
		c.wg.Add(1)
		go func(r *kafka.Reader) {
			defer c.wg.Done()
			c.run(ctx, r)
		}(reader)
	}
	c.wg.Wait()
}

func (c *ChatConsumer) validatePartitionLayout(ctx context.Context) error {
	if c.expectedPartitions <= 0 {
		return nil
	}
	transport, err := kafkautil.Transport(c.conf)
	if err != nil {
		return err
	}
	defer transport.CloseIdleConnections()
	client := &kafka.Client{Addr: kafka.TCP(c.conf.Brokers...), Transport: transport}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	metadata, err := client.Metadata(checkCtx, &kafka.MetadataRequest{Topics: []string{c.conf.Topic}})
	if err != nil {
		return err
	}
	actual := 0
	for _, topic := range metadata.Topics {
		if topic.Name == c.conf.Topic {
			if topic.Error != nil {
				return topic.Error
			}
			actual = len(topic.Partitions)
		}
	}
	if actual < c.expectedPartitions {
		return fmt.Errorf("topic %s has %d partitions, expected at least %d", c.conf.Topic, actual, c.expectedPartitions)
	}
	if c.conf.Consumers > actual {
		logx.Infof("Kafka chat consumers=%d exceeds partitions=%d; extra readers will remain idle", c.conf.Consumers, actual)
	}
	return nil
}

func (c *ChatConsumer) Stop() {
	c.stopOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		for _, reader := range c.readers {
			_ = reader.Close()
		}
		c.wg.Wait()
		if c.dlq != nil {
			_ = c.dlq.Close()
		}
	})
}

func newReader(c kq.KqConf) (*kafka.Reader, error) {
	dialer, err := kafkautil.Dialer(c)
	if err != nil {
		return nil, err
	}
	// Keep the reader's transport settings aligned with the existing kq config.
	if dialer.TLS != nil && dialer.TLS.MinVersion < tls.VersionTLS12 {
		dialer.TLS.MinVersion = tls.VersionTLS12
	}
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.Brokers, GroupID: c.Group, Topic: c.Topic,
		StartOffset: offset(c.Offset), MinBytes: c.MinBytes, MaxBytes: c.MaxBytes,
		MaxWait: time.Second, CommitInterval: 0, QueueCapacity: 1000, Dialer: dialer,
	}), nil
}

func offset(value string) int64 {
	if value == "first" {
		return kafka.FirstOffset
	}
	return kafka.LastOffset
}

func (c *ChatConsumer) run(ctx context.Context, reader *kafka.Reader) {
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return
			}
			logx.Errorf("fetch Kafka chat message: %v", err)
			continue
		}
		handledErr := c.handle(ctx, message)
		if handledErr != nil {
			if c.messageRetry && c.dlq != nil {
				if err := c.publishUntilAccepted(ctx, message, handledErr); err != nil {
					return
				}
			} else {
				logx.Errorf("chat message was not handled: %v", handledErr)
				// Preserve the legacy ForceCommit behavior when retry is disabled.
				if !c.conf.ForceCommit {
					return
				}
			}
		}
		for {
			if err := reader.CommitMessages(ctx, message); err != nil {
				logx.Errorf("commit Kafka chat message: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(250 * time.Millisecond):
				}
				continue
			}
			break
		}
	}
}

func (c *ChatConsumer) handle(ctx context.Context, message kafka.Message) error {
	if !c.messageRetry {
		started := time.Now()
		err := c.handler.Consume(ctx, string(message.Key), string(message.Value))
		c.metrics.ObserveProcessing(time.Since(started), err)
		return err
	}
	return c.policy.Run(ctx, func(_ int) error {
		started := time.Now()
		err := c.handler.Consume(ctx, string(message.Key), string(message.Value))
		c.metrics.ObserveProcessing(time.Since(started), err)
		return err
	}, c.metrics.IncRetry)
}

func (c *ChatConsumer) publishUntilAccepted(ctx context.Context, message kafka.Message, cause error) error {
	record := dlq.NewRecord(c.conf.Topic, string(message.Key), string(message.Value), cause, c.policy.MaxAttempts)
	for {
		err := c.dlq.Publish(ctx, string(message.Key), record)
		if err == nil {
			c.metrics.IncDeadLetter()
			return nil
		}
		logx.Errorf("publish Kafka dead letter: %v", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

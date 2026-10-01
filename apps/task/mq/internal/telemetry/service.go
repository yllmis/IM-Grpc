package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/IM_System/apps/task/mq/internal/kafkautil"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

type Service struct {
	cancel    context.CancelFunc
	client    *kafka.Client
	transport *kafka.Transport
	conf      kq.KqConf
	interval  time.Duration
	metrics   *Metrics
	server    *http.Server
	listener  net.Listener
	stopOnce  sync.Once
}

func NewService(conf kq.KqConf, listenOn string, interval time.Duration, metrics *Metrics) (*Service, error) {
	transport, err := kafkautil.Transport(conf)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", listenOn)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("listen for task-mq metrics: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return &Service{
		client:    &kafka.Client{Addr: kafka.TCP(conf.Brokers...), Transport: transport},
		transport: transport,
		conf:      conf,
		interval:  interval,
		metrics:   metrics,
		server:    &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second},
		listener:  listener,
	}, nil
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go func() {
		if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Errorf("task-mq metrics server: %v", err)
		}
	}()
	s.collect(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx)
		}
	}
}

func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
		s.transport.CloseIdleConnections()
	})
}

func (s *Service) collect(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	lags, err := CollectLag(ctx, s.client, s.conf.Topic, s.conf.Group)
	if err != nil {
		s.metrics.IncLagError()
		logx.Errorf("collect Kafka lag: %v", err)
		return
	}
	for partition, lag := range lags {
		s.metrics.SetLag(s.conf.Topic, s.conf.Group, partition, lag)
	}
}

type lagClient interface {
	Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error)
	ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
	OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
}

func CollectLag(ctx context.Context, client lagClient, topic, group string) (map[int]int64, error) {
	metadata, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
	if err != nil {
		return nil, err
	}
	var partitions []int
	for _, t := range metadata.Topics {
		if t.Name != topic {
			continue
		}
		if t.Error != nil {
			return nil, t.Error
		}
		for _, p := range t.Partitions {
			partitions = append(partitions, p.ID)
		}
	}
	if len(partitions) == 0 {
		return nil, fmt.Errorf("topic %q has no partitions", topic)
	}
	sort.Ints(partitions)
	requests := make([]kafka.OffsetRequest, 0, len(partitions))
	for _, partition := range partitions {
		requests = append(requests, kafka.LastOffsetOf(partition))
	}
	ends, err := client.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Topics: map[string][]kafka.OffsetRequest{topic: requests},
	})
	if err != nil {
		return nil, err
	}
	commits, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
		GroupID: group, Topics: map[string][]int{topic: partitions},
	})
	if err != nil {
		return nil, err
	}
	if commits.Error != nil {
		return nil, commits.Error
	}
	endByPartition := make(map[int]int64, len(partitions))
	for _, p := range ends.Topics[topic] {
		if p.Error != nil {
			return nil, p.Error
		}
		endByPartition[p.Partition] = p.LastOffset
	}
	committedByPartition := make(map[int]int64, len(partitions))
	for _, p := range commits.Topics[topic] {
		if p.Error != nil {
			return nil, p.Error
		}
		committedByPartition[p.Partition] = p.CommittedOffset
	}
	lags := make(map[int]int64, len(partitions))
	for _, partition := range partitions {
		committed := committedByPartition[partition]
		if committed < 0 {
			committed = 0
		}
		lag := endByPartition[partition] - committed
		if lag < 0 {
			lag = 0
		}
		lags[partition] = lag
	}
	return lags, nil
}

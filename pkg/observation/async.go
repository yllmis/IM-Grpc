package observation

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// AsyncObservationSink 包装任意 Sink，缓冲后后台写入。
// 缓冲区满时丢弃事件并累计缺口，后续成功写入时补 observation_gap 标记。
// Record 永不阻塞调用方业务路径。
type AsyncObservationSink struct {
	inner  ObservationSink
	ch     chan MessageEvent
	source string

	metrics Metrics

	closeOnce sync.Once
	closed    atomic.Bool
	done      chan struct{}
	wg        sync.WaitGroup

	mu          sync.Mutex
	dropped     int64
	gapStartAt  int64
	gapEndAt    int64
	gapReason   string
	gapPending  bool
	lastPersist time.Time
}

// NewAsyncObservationSink 创建异步包装层。bufferSize<=0 时用 1024。
func NewAsyncObservationSink(inner ObservationSink, source string, bufferSize int) *AsyncObservationSink {
	if bufferSize <= 0 {
		bufferSize = 1024
	}
	if source == "" {
		source = SourceImWs
	}

	s := &AsyncObservationSink{
		inner:  inner,
		ch:     make(chan MessageEvent, bufferSize),
		source: source,
		done:   make(chan struct{}),
	}

	s.wg.Add(1)
	go s.loop()
	return s
}

// Metrics 暴露内部计数器。
func (s *AsyncObservationSink) Metrics() *Metrics { return &s.metrics }

// Record 非阻塞投递。缓冲区满时丢弃并累计观测缺口。
func (s *AsyncObservationSink) Record(_ context.Context, event MessageEvent) error {
	if s.closed.Load() {
		s.noteDrop(event.OccurredAt, "sink_closed")
		return nil
	}

	if event.EventVersion == 0 {
		event.EventVersion = EventVersion
	}

	select {
	case s.ch <- event:
		s.metrics.Recorded.Add(1)
		return nil
	default:
		s.noteDrop(event.OccurredAt, "buffer_full")
		s.metrics.Dropped.Add(1)
		return nil
	}
}

func (s *AsyncObservationSink) noteDrop(at int64, reason string) {
	if at <= 0 {
		at = NowUnixNano()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropped++
	s.gapPending = true
	if s.gapStartAt == 0 || at < s.gapStartAt {
		s.gapStartAt = at
	}
	if at > s.gapEndAt {
		s.gapEndAt = at
	}
	s.gapReason = reason
}

func (s *AsyncObservationSink) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			s.drain()
			s.flushGap()
			return
		case ev := <-s.ch:
			s.writeEvent(ev)
			s.maybeFlushGap()
		case <-ticker.C:
			s.maybeFlushGap()
		}
	}
}

func (s *AsyncObservationSink) writeEvent(ev MessageEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.inner.Record(ctx, ev); err != nil {
		s.metrics.Failed.Add(1)
		s.noteDrop(ev.OccurredAt, "write_failed")
	}
}

func (s *AsyncObservationSink) drain() {
	for {
		select {
		case ev := <-s.ch:
			s.writeEvent(ev)
		default:
			return
		}
	}
}

func (s *AsyncObservationSink) maybeFlushGap() {
	s.mu.Lock()
	if !s.gapPending || s.dropped == 0 {
		s.mu.Unlock()
		return
	}
	// 节流：距上次 gap 标记至少 1s，避免刷屏
	if !s.lastPersist.IsZero() && time.Since(s.lastPersist) < time.Second {
		s.mu.Unlock()
		return
	}
	gap := NewObservationGap(s.source, s.dropped, s.gapStartAt, s.gapEndAt, s.gapReason)
	s.gapPending = false
	s.lastPersist = time.Now()
	// 保留计数直到写入成功？契约要求尽力写入；这里重置范围、累计计数已写入事件
	s.dropped = 0
	s.gapStartAt = 0
	s.gapEndAt = 0
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.inner.Record(ctx, gap); err != nil {
		s.metrics.Failed.Add(1)
		// 标记失败：查询侧必须将覆盖视为 unknown（无 gap 记录时的兜底）
	} else {
		s.metrics.GapMarkers.Add(1)
	}
}

func (s *AsyncObservationSink) flushGap() {
	s.mu.Lock()
	if !s.gapPending || s.dropped == 0 {
		s.mu.Unlock()
		return
	}
	gap := NewObservationGap(s.source, s.dropped, s.gapStartAt, s.gapEndAt, s.gapReason)
	s.gapPending = false
	s.dropped = 0
	s.gapStartAt = 0
	s.gapEndAt = 0
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.inner.Record(ctx, gap); err == nil {
		s.metrics.GapMarkers.Add(1)
	}
}

// Close 尽力 flush 后关闭。flush 失败允许丢事件（宁丢事件不丢消息）。
func (s *AsyncObservationSink) Close() {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		close(s.done)
		s.wg.Wait()
	})
}

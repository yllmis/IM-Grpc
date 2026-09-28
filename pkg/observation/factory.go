package observation

import (
	"context"
	"fmt"
)

// NewSink 按配置构建观测 Sink。
// Enabled=false → Noop（零行为变化）。
// Enabled=true 且 PersistEvents → Async(Mongo)。
func NewSink(ctx context.Context, cfg DeliveryObservation, url, db string) (ObservationSink, func(), error) {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	if !cfg.Enabled {
		return NoopObservationSink{}, func() {}, nil
	}

	if !cfg.PersistEvents {
		return nil, nil, fmt.Errorf("observation: Enabled=true requires PersistEvents=true")
	}

	mongoSink, err := NewMongoObservationSink(ctx, url, db)
	if err != nil {
		return nil, nil, err
	}

	async := NewAsyncObservationSink(mongoSink, SourceImWs, cfg.BufferSize)
	return async, async.Close, nil
}

// NewSinkWithSource 同 NewSink，但指定事件 source（im-ws / task-mq / mq-push ...）。
func NewSinkWithSource(ctx context.Context, cfg DeliveryObservation, url, db, source string) (ObservationSink, func(), error) {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	if !cfg.Enabled {
		return NoopObservationSink{}, func() {}, nil
	}

	mongoSink, err := NewMongoObservationSink(ctx, url, db)
	if err != nil {
		return nil, nil, err
	}
	async := NewAsyncObservationSink(mongoSink, source, cfg.BufferSize)
	return async, async.Close, nil
}

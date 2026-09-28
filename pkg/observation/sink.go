package observation

import "context"

// ObservationSink 旁路记录观测事件。
// Record 不得阻塞业务主路径；实现失败只记日志/指标，不影响调用方返回值。
type ObservationSink interface {
	Record(ctx context.Context, event MessageEvent) error
}

// NoopObservationSink 丢弃所有事件。观测关闭时的默认实现。
type NoopObservationSink struct{}

func (NoopObservationSink) Record(context.Context, MessageEvent) error { return nil }

// Nop 返回默认 Noop sink。
func Nop() ObservationSink { return NoopObservationSink{} }

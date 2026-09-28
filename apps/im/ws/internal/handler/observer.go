package handler

import (
	"context"
	"fmt"

	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/pkg/observation"
)

// SinkObserver 将 WS 连接/ACK 旁路事件写入 ObservationSink。
// 永不反向影响连接与 ACK 重试语义。
type SinkObserver struct {
	sink       observation.ObservationSink
	instanceID string
}

var _ websocket.Observer = (*SinkObserver)(nil)

func NewSinkObserver(sink observation.ObservationSink, instanceID string) *SinkObserver {
	if sink == nil {
		sink = observation.Nop()
	}
	return &SinkObserver{sink: sink, instanceID: instanceID}
}

func (o *SinkObserver) OnConnect(uid, connectionID string) {
	observation.SafeRecord(context.Background(), o.sink, observation.ConnectionEvent(
		observation.EventOnline, observation.SourceImWs, uid, connectionID, o.instanceID, "connect",
	))
}

func (o *SinkObserver) OnDisconnect(uid, connectionID, reason string) {
	observation.SafeRecord(context.Background(), o.sink, observation.ConnectionEvent(
		observation.EventOffline, observation.SourceImWs, uid, connectionID, o.instanceID, reason,
	))
}

func (o *SinkObserver) OnAckReceived(uid, frameID string, ackSeq int) {
	observation.SafeRecord(context.Background(), o.sink, ackEvent(
		observation.EventAckReceived, uid, frameID, ackSeq, "",
	))
}

func (o *SinkObserver) OnAckTimeout(uid, frameID string, ackSeq int) {
	observation.SafeRecord(context.Background(), o.sink, ackEvent(
		observation.EventAckTimeout, uid, frameID, ackSeq, observation.ErrCodeAckTimeout,
	))
}

// ackEvent 传输层 ACK 事件。
// WS 帧 Id 是客户端消息 ID；messageId 必填约束对 ack 放宽为 frameID 兜底（契约 3.1）。
func ackEvent(eventType, uid, frameID string, ackSeq int, errCode string) observation.MessageEvent {
	return observation.MessageEvent{
		EventID:         observation.NewEventID(),
		EventVersion:    observation.EventVersion,
		EventType:       eventType,
		MessageID:       frameID,
		ClientMessageID: frameID,
		SenderID:        uid,
		OccurredAt:      observation.NowUnixNano(),
		Source:          observation.SourceImWs,
		ErrorCode:       errCode,
		Sequence:        int64(ackSeq),
		Metadata: map[string]string{
			"transport": "ws-ack",
			"ackSeq":    fmt.Sprintf("%d", ackSeq),
		},
	}
}

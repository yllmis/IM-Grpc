package observation

import "context"

// SafeRecord 旁路记录事件，永不向业务返回错误。
func SafeRecord(ctx context.Context, sink ObservationSink, event MessageEvent) {
	if sink == nil {
		return
	}
	if err := sink.Record(ctx, event); err != nil {
		// 契约：观测失败不影响业务；详细错误由 Sink 内部记日志/指标
		return
	}
}

// MessageEventBuilder 构造消息生命周期事件的公共字段。
type MessageEventBuilder struct {
	MessageID       string
	ClientMessageID string
	CorrelationID   string
	ConversationID  string
	SenderID        string
	ReceiverID      string
	AttemptID       string
	Source          string
}

func (b MessageEventBuilder) build(eventType string) MessageEvent {
	corr := b.CorrelationID
	if corr == "" {
		corr = b.MessageID
	}
	return MessageEvent{
		EventID:         NewEventID(),
		EventVersion:    EventVersion,
		EventType:       eventType,
		MessageID:       b.MessageID,
		ClientMessageID: b.ClientMessageID,
		CorrelationID:   corr,
		ConversationID:  b.ConversationID,
		SenderID:        b.SenderID,
		ReceiverID:      b.ReceiverID,
		AttemptID:       b.AttemptID,
		OccurredAt:      NowUnixNano(),
		Source:          b.Source,
	}
}

func (b MessageEventBuilder) buildErr(eventType, errorCode string) MessageEvent {
	ev := b.build(eventType)
	ev.ErrorCode = errorCode
	return ev
}

// 生命周期事件构造器
func (b MessageEventBuilder) Accepted() MessageEvent {
	return b.build(EventAccepted)
}

func (b MessageEventBuilder) KafkaPublished() MessageEvent {
	return b.build(EventKafkaPublished)
}

func (b MessageEventBuilder) KafkaPublishFailed(errCode string) MessageEvent {
	return b.buildErr(EventKafkaPublishFailed, errCode)
}

func (b MessageEventBuilder) KafkaConsumed() MessageEvent {
	return b.build(EventKafkaConsumed)
}

func (b MessageEventBuilder) Persisted() MessageEvent {
	return b.build(EventPersisted)
}

func (b MessageEventBuilder) PersistFailed(errCode string) MessageEvent {
	return b.buildErr(EventPersistFailed, errCode)
}

func (b MessageEventBuilder) DeliveryAttempted() MessageEvent {
	return b.build(EventDeliveryAttempted)
}

func (b MessageEventBuilder) DeliverySucceeded() MessageEvent {
	return b.build(EventDeliverySucceeded)
}

func (b MessageEventBuilder) DeliveryFailed(errCode string) MessageEvent {
	return b.buildErr(EventDeliveryFailed, errCode)
}

func (b MessageEventBuilder) ReceiverOffline() MessageEvent {
	return b.buildErr(EventReceiverOffline, ErrCodeDeliveryNoConn)
}

func (b MessageEventBuilder) ReadConfirmed() MessageEvent {
	return b.build(EventReadConfirmed)
}

// ConnectionEvent 构造 online/offline 事件（messageId 可空）。
func ConnectionEvent(eventType, source, userID, connectionID, instanceID, reason string) MessageEvent {
	return MessageEvent{
		EventID:      NewEventID(),
		EventVersion: EventVersion,
		EventType:    eventType,
		SenderID:     userID,
		OccurredAt:   NowUnixNano(),
		Source:       source,
		Metadata: map[string]string{
			"connectionId": connectionID,
			"instanceId":   instanceID,
			"reason":       reason,
		},
	}
}

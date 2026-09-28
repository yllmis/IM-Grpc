// Package observation 提供消息生命周期观测事件的共享契约与旁路写入能力。
// 写侧（im-ws / task-mq / mqclient）与读侧（operations）共用本包类型。
// 语义契约见 docs/message-event-contract.md。
package observation

// EventVersion 当前事件语义版本。语义变更时递增。
const EventVersion int32 = 1

// 事件类型。只允许追加；禁止改既有含义（须升 EventVersion 并更新文档）。
const (
	// 消息生命周期
	EventAccepted           = "accepted"
	EventKafkaPublished     = "kafka_published"
	EventKafkaPublishFailed = "kafka_publish_failed"
	EventKafkaConsumed      = "kafka_consumed"
	EventPersisted          = "persisted"
	EventPersistFailed      = "persist_failed"
	EventDeliveryAttempted  = "delivery_attempted"
	EventDeliverySucceeded  = "delivery_succeeded"
	EventDeliveryFailed     = "delivery_failed"
	EventReceiverOffline    = "receiver_offline"
	EventAckReceived        = "ack_received"
	EventAckTimeout         = "ack_timeout"
	EventReadConfirmed      = "read_confirmed"

	// 连接观察
	EventOnline  = "online"
	EventOffline = "offline"

	// 观测缺口标记：异步丢弃后尽力写入，告知查询侧覆盖不完整
	EventObservationGap = "observation_gap"
)

// source 取值
const (
	SourceImWs         = "im-ws"
	SourceTaskMq       = "task-mq"
	SourceMqPush       = "mq-push"
	SourceWsPush       = "ws-push"
	SourceOperations   = "operations"
	SourceCompatLegacy = "compat-legacy"
)

// errorCode 取值（事件内业务枚举，与 gRPC 状态码无关）
const (
	ErrCodeKafkaPublishFailed  = "KAFKA_PUBLISH_FAILED"
	ErrCodePersistFailed       = "PERSIST_FAILED"
	ErrCodeDeliveryWriteFailed = "DELIVERY_WRITE_FAILED"
	ErrCodeDeliveryNoConn      = "DELIVERY_NO_CONN"
	ErrCodeAckTimeout          = "ACK_TIMEOUT"
	ErrCodeInvalidEvent        = "INVALID_EVENT"
)

// MessageEvent 是 message_events 集合的单条事件。
// 禁止写入消息正文、Token、密码、手机号、原始 bitmap / payload。
type MessageEvent struct {
	EventID         string            `bson:"eventId"`
	EventVersion    int32             `bson:"eventVersion"`
	EventType       string            `bson:"eventType"`
	MessageID       string            `bson:"messageId,omitempty"`
	ClientMessageID string            `bson:"clientMessageId,omitempty"`
	CorrelationID   string            `bson:"correlationId,omitempty"`
	ConversationID  string            `bson:"conversationId,omitempty"`
	SenderID        string            `bson:"senderId,omitempty"`
	ReceiverID      string            `bson:"receiverId,omitempty"`
	AttemptID       string            `bson:"attemptId,omitempty"`
	OccurredAt      int64             `bson:"occurredAt"` // UnixNano
	Source          string            `bson:"source"`
	ErrorCode       string            `bson:"errorCode,omitempty"`
	Sequence        int64             `bson:"sequence,omitempty"`
	Metadata        map[string]string `bson:"metadata,omitempty"`
}

// NewEventID 生成事件唯一 ID（24hex ObjectID）。
func NewEventID() string {
	return newObjectIDHex()
}

// Validate 校验必填字段。非法事件应被拒绝并记 INVALID_EVENT，不得进入存储。
func (e MessageEvent) Validate() error {
	if e.EventID == "" {
		return errInvalidEvent("eventId is required")
	}
	if e.EventType == "" {
		return errInvalidEvent("eventType is required")
	}
	if e.Source == "" {
		return errInvalidEvent("source is required")
	}
	if e.OccurredAt <= 0 {
		return errInvalidEvent("occurredAt must be UnixNano > 0")
	}
	// 消息生命周期事件必须携带 messageId；连接事件与 observation_gap 允许为空
	switch e.EventType {
	case EventOnline, EventOffline, EventObservationGap:
	default:
		if e.MessageID == "" {
			return errInvalidEvent("messageId is required for event type " + e.EventType)
		}
	}
	return nil
}

type invalidEventError struct{ msg string }

func (e invalidEventError) Error() string { return "observation: invalid event: " + e.msg }

func errInvalidEvent(msg string) error { return invalidEventError{msg: msg} }

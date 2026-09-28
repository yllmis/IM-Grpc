package logic

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"github.com/IM_System/pkg/observation"
)

func nowUnixNano() int64 { return time.Now().UnixNano() }

// EncodeCursor 生成不透明游标 base64(occurredAt:eventId)。
func EncodeCursor(occurredAt int64, eventID string) string {
	raw := fmt.Sprintf("%d:%s", occurredAt, eventID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor 解析游标。非法游标 → INVALID_ARGUMENT。
func DecodeCursor(cursor string) (int64, string, error) {
	if cursor == "" {
		return 0, "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", types.InvalidArgument("invalid cursor")
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return 0, "", types.InvalidArgument("invalid cursor")
	}
	var occurredAt int64
	if _, err := fmt.Sscanf(parts[0], "%d", &occurredAt); err != nil {
		return 0, "", types.InvalidArgument("invalid cursor")
	}
	if parts[1] == "" {
		return 0, "", types.InvalidArgument("invalid cursor")
	}
	return occurredAt, parts[1], nil
}

// toProtoEvent 转换事件，原样返回未知 eventType。
func toProtoEvent(ev *observation.MessageEvent) *operations.MessageEvent {
	if ev == nil {
		return nil
	}
	md := ev.Metadata
	if md == nil {
		md = map[string]string{}
	}
	return &operations.MessageEvent{
		EventId:         ev.EventID,
		EventVersion:    ev.EventVersion,
		EventType:       ev.EventType,
		MessageId:       ev.MessageID,
		ClientMessageId: ev.ClientMessageID,
		CorrelationId:   ev.CorrelationID,
		ConversationId:  ev.ConversationID,
		SenderId:        ev.SenderID,
		ReceiverId:      ev.ReceiverID,
		AttemptId:       ev.AttemptID,
		OccurredAt:      ev.OccurredAt,
		Source:          ev.Source,
		ErrorCode:       ev.ErrorCode,
		Sequence:        ev.Sequence,
		Metadata:        md,
	}
}

// evidenceOf 投递事件证据强度，强制如实标注。
func evidenceOf(eventType string) string {
	switch eventType {
	case observation.EventDeliverySucceeded, observation.EventDeliveryAttempted, observation.EventDeliveryFailed:
		return "gateway-write"
	case observation.EventAckReceived, observation.EventAckTimeout:
		return "transport-ack"
	case observation.EventReceiverOffline:
		return "offline-marker"
	default:
		return "legacy"
	}
}

// deliveryEventTypes 投递侧事件类型集合。
func deliveryEventTypes() []string {
	return []string{
		observation.EventDeliveryAttempted,
		observation.EventDeliverySucceeded,
		observation.EventDeliveryFailed,
		observation.EventReceiverOffline,
		observation.EventAckReceived,
		observation.EventAckTimeout,
	}
}

// windowCovered 判断观测窗口是否可证明完整。
// 观测上线前的历史无法证明；显式查询很早的时间窗视为未覆盖。
func windowCovered(startTime, endTime, observedAt int64) bool {
	_ = endTime
	_ = observedAt
	const rolloutCutoff = int64(1767225600000000000) // 2026-01-01 UTC 纳秒
	if startTime > 0 && startTime < rolloutCutoff {
		return false
	}
	return true
}

// coverageFrom 查询 observation_gap 并给出覆盖状态。
func coverageFrom(ctx context.Context, q operationsmodels.EventModel, start, end, observedAt int64) (string, uint64, error) {
	dropped, found, err := q.CountGaps(ctx, start, end)
	if err != nil {
		return types.CoverageUnknown, 0, types.MapQueryError(err)
	}
	st, d := types.ResolveCoverage(dropped, found, windowCovered(start, end, observedAt))
	return st, d, nil
}

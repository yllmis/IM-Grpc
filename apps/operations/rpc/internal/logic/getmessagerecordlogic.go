package logic

import (
	"context"

	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/operations/rpc/internal/faultinject"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/serviceauth"
)

type GetMessageRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMessageRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageRecordLogic {
	return &GetMessageRecordLogic{ctx: ctx, svcCtx: svcCtx}
}

// GetMessageRecord 单条消息元数据摘要（不含正文）。
// found=false 表示查询成功但无记录；超时/DB 错误必须走 gRPC 状态，禁止变成 found=false。
func (l *GetMessageRecordLogic) GetMessageRecord(in *operations.GetMessageRecordRequest) (*operations.GetMessageRecordResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if err := types.ValidateMessageID(in.MessageId); err != nil {
		return nil, err
	}

	observedAt := nowUnixNano()
	if response, injected, err := l.injectedMessageRecord(in.MessageId, observedAt); injected {
		return response, err
	}

	if l.svcCtx.MessageQueryRpc == nil {
		return nil, types.Unimplemented("message domain query unavailable")
	}
	row, err := l.svcCtx.MessageQueryRpc.GetMessageRecord(serviceauth.Outgoing(l.ctx, l.svcCtx.Config.DomainQueryToken), &im.MessageRecordRequest{MessageId: in.MessageId})
	if err != nil {
		return nil, types.MapQueryError(err)
	}
	if row == nil {
		return nil, types.Internal("empty message domain response")
	}
	if !row.Found {
		return l.notFound(in.MessageId, row.ObservedAt)
	}

	eventsAvailable := false
	if l.svcCtx.EventModel != nil {
		var evErr error
		eventsAvailable, evErr = l.svcCtx.EventModel.HasAnyByMessageID(l.ctx, in.MessageId)
		if evErr != nil {
			// 业务记录已成功查询，观测失败不得将它变成查询失败。
			return messageRecordResponse(row, false, "event-availability-unknown"), nil
		}
	}

	note := ""
	if !eventsAvailable {
		note = "no-events-yet"
	}

	if l.svcCtx.EventModel == nil {
		note = "event-availability-unknown"
	}
	return messageRecordResponse(row, eventsAvailable, note), nil
}

func messageRecordResponse(row *im.MessageRecordResponse, available bool, note string) *operations.GetMessageRecordResponse {
	state := "absent"
	if available {
		state = "available"
	}
	if note == "event-availability-unknown" {
		state = "unknown"
	}
	return &operations.GetMessageRecordResponse{Found: row.Found, MessageId: row.MessageId, ConversationId: row.ConversationId,
		SenderId: row.SenderId, ReceiverId: row.ReceiverId, CreatedAt: row.CreatedAt, Source: "chat_log", ObservedAt: row.ObservedAt,
		ChatType: row.ChatType, MsgType: row.MsgType, ReadState: row.ReadState, ReadStateNote: row.ReadStateNote,
		EventsAvailable: available, EventsState: state, Note: note}
}

// injectedMessageRecord is a test-only response switch. It runs before Mongo
// and never writes data; with FaultInjection disabled it is a no-op.
func (l *GetMessageRecordLogic) injectedMessageRecord(messageID string, observedAt int64) (*operations.GetMessageRecordResponse, bool, error) {
	scenario, ok := l.svcCtx.FaultInjection.ScenarioFor(messageID)
	if !ok {
		return nil, false, nil
	}
	switch scenario {
	case faultinject.MessageMissing:
		// 合成空结果不查询 Mongo；来源明确标记，避免被当作真实存储故障。
		return &operations.GetMessageRecordResponse{
			Found: false, MessageId: messageID, ObservedAt: observedAt,
			Source: "fault-injection", Note: "injected-query-succeeded-no-record",
		}, true, nil
	case faultinject.QueryTimeout:
		return nil, true, types.MapQueryError(context.DeadlineExceeded)
	case faultinject.PermissionDenied:
		return nil, true, types.PermissionDenied("fault injection permission denied")
	case faultinject.UnsupportedCapability:
		return nil, true, types.Unimplemented("fault injection unsupported capability")
	case faultinject.WrongMessageID:
		return &operations.GetMessageRecordResponse{
			Found:      true,
			MessageId:  "000000000000000000000001",
			CreatedAt:  observedAt,
			Source:     "fault-injection",
			ObservedAt: observedAt,
			Note:       "injected-wrong-message-id",
		}, true, nil
	case faultinject.MalformedResponse:
		return &operations.GetMessageRecordResponse{
			Found:     true,
			MessageId: messageID,
			Source:    "fault-injection",
			Note:      "injected-malformed-observation",
		}, true, nil
	default:
		return nil, false, nil
	}
}

func (l *GetMessageRecordLogic) notFound(messageId string, observedAt int64) (*operations.GetMessageRecordResponse, error) {
	resp := &operations.GetMessageRecordResponse{
		Found:       false,
		MessageId:   messageId,
		ObservedAt:  observedAt,
		Source:      "chat_log",
		Note:        "query-succeeded-no-record",
		EventsState: "unknown",
	}
	// 仅有事件、ChatLog 未找到 → events-only（异常态）
	if l.svcCtx.EventModel != nil {
		if ok, err := l.svcCtx.EventModel.HasAnyByMessageID(l.ctx, messageId); err != nil {
			resp.Note = joinNote(resp.Note, "event-availability-unknown")
		} else if ok {
			resp.EventsAvailable = true
			resp.EventsState = "available"
			resp.Source = "events-only"
			resp.Note = "events-only-chatlog-missing"
		} else {
			resp.EventsState = "absent"
		}
	}
	return resp, nil
}

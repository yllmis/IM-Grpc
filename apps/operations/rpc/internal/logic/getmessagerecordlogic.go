package logic

import (
	"context"
	"errors"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/constants"
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

	chatLog, err := l.svcCtx.ChatLogModel.FindOne(l.ctx, in.MessageId)
	if err != nil {
		if errors.Is(err, immodels.ErrNotFound) {
			return l.notFound(in.MessageId, observedAt)
		}
		if errors.Is(err, immodels.ErrInvalidObjectId) {
			return nil, types.InvalidArgument("messageId must be 24hex ObjectID")
		}
		return nil, types.MapQueryError(err)
	}

	eventsAvailable := false
	if l.svcCtx.EventModel != nil {
		var evErr error
		eventsAvailable, evErr = l.svcCtx.EventModel.HasAnyByMessageID(l.ctx, in.MessageId)
		if evErr != nil {
			return nil, types.MapQueryError(evErr)
		}
	}

	note := ""
	if !eventsAvailable {
		note = "no-events-yet"
	}

	readState, readNote := deriveReadState(chatLog)

	return &operations.GetMessageRecordResponse{
		Found:           true,
		MessageId:       chatLog.ID.Hex(),
		ConversationId:  chatLog.ConversationId,
		SenderId:        chatLog.SendId,
		ReceiverId:      chatLog.RecvId,
		CreatedAt:       chatLog.SendTime,
		Source:          "chat_log",
		ObservedAt:      observedAt,
		ChatType:        int32(chatLog.ChatType),
		MsgType:         int32(chatLog.MsgType),
		ReadState:       readState,
		ReadStateNote:   readNote,
		EventsAvailable: eventsAvailable,
		Note:            note,
	}, nil
}

func (l *GetMessageRecordLogic) notFound(messageId string, observedAt int64) (*operations.GetMessageRecordResponse, error) {
	resp := &operations.GetMessageRecordResponse{
		Found:      false,
		MessageId:  messageId,
		ObservedAt: observedAt,
		Source:     "chat_log",
		Note:       "query-succeeded-no-record",
	}
	// 仅有事件、ChatLog 未找到 → events-only（异常态）
	if l.svcCtx.EventModel != nil {
		if ok, err := l.svcCtx.EventModel.HasAnyByMessageID(l.ctx, messageId); err == nil && ok {
			resp.EventsAvailable = true
			resp.Source = "events-only"
			resp.Note = "events-only-chatlog-missing"
		}
	}
	return resp, nil
}

// deriveReadState 已读摘要。
// 不返回 bitmap 与用户列表（ADR-008）；群聊 bitmap 哈希碰撞只能是 approximate。
func deriveReadState(chatLog *immodels.ChatLog) (string, string) {
	if chatLog == nil {
		return "unknown", "no-record"
	}

	switch constants.ChatType(chatLog.ChatType) {
	case constants.SingleChatType:
		if len(chatLog.ReadRecords) == 0 {
			return "unknown", "empty-read-records"
		}
		if chatLog.ReadRecords[0] == 0 {
			return "known", "single-chat-unread"
		}
		return "known", "single-chat-read"
	case constants.GroupChatType:
		if len(chatLog.ReadRecords) == 0 {
			return "unknown", "empty-read-records"
		}
		return "approximate", "group-bitmap-hash-collision-risk"
	default:
		return "unknown", "unsupported-chat-type"
	}
}

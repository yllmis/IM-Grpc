package logic

import (
	"context"
	"strings"
	"time"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
)

type SearchMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchMessagesLogic {
	return &SearchMessagesLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *SearchMessagesLogic) SearchMessages(in *operations.SearchMessagesRequest) (*operations.SearchMessagesResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if strings.TrimSpace(in.SenderId) == "" || len(in.SenderId) > 128 || len(in.ReceiverId) > 128 || strings.ContainsAny(in.SenderId+in.ReceiverId, "\x00\n\r") {
		return nil, types.InvalidArgument("valid senderId is required")
	}
	// 时间范围必须明确：禁止无界扫描，也不把缺少时间解释为查询空结果。
	if in.StartTime <= 0 || in.EndTime <= in.StartTime || in.EndTime-in.StartTime > int64(7*24*time.Hour) {
		return nil, types.InvalidArgument("positive time range must be <= 7 days")
	}
	limit, err := types.ValidateLimit(in.Limit, 10, 20)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.MessageSearch == nil {
		return nil, types.Unimplemented("message search unavailable")
	}
	rows, err := l.svcCtx.MessageSearch.Search(l.ctx, operationsmodels.MessageSearchFilter{SenderID: in.SenderId, ReceiverID: in.ReceiverId, StartTime: in.StartTime, EndTime: in.EndTime, Limit: limit})
	if err != nil {
		return nil, types.MapQueryError(err)
	}
	truncated := len(rows) > int(limit)
	if truncated {
		rows = rows[:limit]
	}
	result := &operations.SearchMessagesResponse{Messages: make([]*operations.MessageReference, 0, len(rows)), Truncated: truncated, ObservedAt: nowUnixNano()}
	for _, row := range rows {
		result.Messages = append(result.Messages, &operations.MessageReference{MessageId: row.ID.Hex(), ConversationId: row.ConversationID, SenderId: row.SenderID, ReceiverId: row.ReceiverID, CreatedAt: row.CreatedAt})
	}
	return result, nil
}

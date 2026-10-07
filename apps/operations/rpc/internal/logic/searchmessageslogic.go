package logic

import (
	"context"

	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/serviceauth"
)

type SearchMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchMessagesLogic(ctx context.Context, s *svc.ServiceContext) *SearchMessagesLogic {
	return &SearchMessagesLogic{ctx: ctx, svcCtx: s}
}

// Compatibility facade: validation, bounded storage access and facts belong to IM.
func (l *SearchMessagesLogic) SearchMessages(in *operations.SearchMessagesRequest) (*operations.SearchMessagesResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if l.svcCtx.MessageQueryRpc == nil {
		return nil, types.Unimplemented("message domain query unavailable")
	}
	resp, err := l.svcCtx.MessageQueryRpc.SearchMessages(serviceauth.Outgoing(l.ctx, l.svcCtx.Config.DomainQueryToken), &im.MessageSearchRequest{
		SenderId: in.SenderId, ReceiverId: in.ReceiverId, StartTime: in.StartTime, EndTime: in.EndTime, Limit: in.Limit,
	})
	if err != nil {
		return nil, types.MapQueryError(err)
	}
	if resp == nil {
		return nil, types.Internal("empty message domain response")
	}
	result := &operations.SearchMessagesResponse{Messages: make([]*operations.MessageReference, 0, len(resp.Messages)), Truncated: resp.Truncated, ObservedAt: resp.ObservedAt}
	for _, row := range resp.Messages {
		if row == nil {
			return nil, types.Internal("invalid message domain response")
		}
		result.Messages = append(result.Messages, &operations.MessageReference{MessageId: row.MessageId, ConversationId: row.ConversationId, SenderId: row.SenderId, ReceiverId: row.ReceiverId, CreatedAt: row.CreatedAt})
	}
	return result, nil
}

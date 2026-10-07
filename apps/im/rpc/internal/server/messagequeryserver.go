package server

import (
	"context"
	"errors"
	"time"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/readquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageQueryServer struct {
	im.UnimplementedMessageQueryServer
	svcCtx *svc.ServiceContext
}

func NewMessageQueryServer(s *svc.ServiceContext) *MessageQueryServer {
	return &MessageQueryServer{svcCtx: s}
}

// Only message facts are queried; observation outages cannot affect this RPC.
func (s *MessageQueryServer) GetMessageRecord(ctx context.Context, in *im.MessageRecordRequest) (*im.MessageRecordResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if err := readquery.ValidateMessageID(in.MessageId); err != nil {
		return nil, err
	}
	if s.svcCtx.ChatLogModel == nil {
		return nil, status.Error(codes.Unavailable, "message query unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, readquery.Timeout)
	defer cancel()
	row, err := s.svcCtx.ChatLogModel.FindOne(ctx, in.MessageId)
	if errors.Is(err, immodels.ErrNotFound) {
		return &im.MessageRecordResponse{MessageId: in.MessageId, ObservedAt: time.Now().UnixNano()}, nil
	}
	if err != nil {
		return nil, readquery.MapError(err)
	}
	if row == nil {
		return nil, status.Error(codes.Internal, "invalid message query result")
	}
	readState, readNote := deriveReadState(row)
	return &im.MessageRecordResponse{
		Found: true, MessageId: row.ID.Hex(), ConversationId: row.ConversationId,
		SenderId: row.SendId, ReceiverId: row.RecvId, CreatedAt: row.SendTime,
		ChatType: int32(row.ChatType), MsgType: int32(row.MsgType),
		ReadState: readState, ReadStateNote: readNote, ObservedAt: time.Now().UnixNano(),
	}, nil
}
func (s *MessageQueryServer) SearchMessages(ctx context.Context, in *im.MessageSearchRequest) (*im.MessageSearchResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if !readquery.ValidIdentifier(in.SenderId) || (in.ReceiverId != "" && !readquery.ValidIdentifier(in.ReceiverId)) {
		return nil, status.Error(codes.InvalidArgument, "valid senderId/receiverId required")
	}
	if in.StartTime <= 0 || in.EndTime <= in.StartTime || in.EndTime-in.StartTime > int64(7*24*time.Hour) {
		return nil, status.Error(codes.InvalidArgument, "positive time range must be <= 7 days")
	}
	limit, err := readquery.Limit(in.Limit, 10, 20)
	if err != nil {
		return nil, err
	}
	if s.svcCtx.MessageSearch == nil {
		return nil, status.Error(codes.Unavailable, "message search unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, readquery.Timeout)
	defer cancel()
	rows, err := s.svcCtx.MessageSearch.Search(ctx, immodels.MessageSearchFilter{
		SenderID: in.SenderId, ReceiverID: in.ReceiverId, StartTime: in.StartTime, EndTime: in.EndTime, Limit: limit,
	})
	if err != nil {
		return nil, readquery.MapError(err)
	}
	truncated := len(rows) > int(limit)
	if truncated {
		rows = rows[:limit]
	}
	resp := &im.MessageSearchResponse{Messages: make([]*im.MessageQueryReference, 0, len(rows)), Truncated: truncated, ObservedAt: time.Now().UnixNano()}
	for _, row := range rows {
		resp.Messages = append(resp.Messages, &im.MessageQueryReference{MessageId: row.ID.Hex(), ConversationId: row.ConversationID, SenderId: row.SenderID, ReceiverId: row.ReceiverID, CreatedAt: row.CreatedAt})
	}
	return resp, nil
}

// Read state belongs to the message domain; no raw bitmap crosses this contract.
func deriveReadState(row *immodels.ChatLog) (string, string) {
	if row == nil {
		return "unknown", "no-record"
	}
	if len(row.ReadRecords) == 0 {
		return "unknown", "empty-read-records"
	}
	switch constants.ChatType(row.ChatType) {
	case constants.SingleChatType:
		if row.ReadRecords[0] == 0 {
			return "known", "single-chat-unread"
		}
		return "known", "single-chat-read"
	case constants.GroupChatType:
		return "approximate", "group-bitmap-hash-collision-risk"
	default:
		return "unknown", "unsupported-chat-type"
	}
}

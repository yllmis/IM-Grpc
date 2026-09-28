package logic

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
)

type GetMessageTimelineLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMessageTimelineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageTimelineLogic {
	return &GetMessageTimelineLogic{ctx: ctx, svcCtx: svcCtx}
}

// GetMessageTimeline 消息生命周期事件流。
// 空结果仅在 coverageStatus=complete 时才表示“窗口内没有已记录事件”。
func (l *GetMessageTimelineLogic) GetMessageTimeline(in *operations.GetMessageTimelineRequest) (*operations.GetMessageTimelineResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if err := types.ValidateMessageID(in.MessageId); err != nil {
		return nil, err
	}
	if err := types.ValidateTimeRange(in.StartTime, in.EndTime); err != nil {
		return nil, err
	}
	limit, err := types.ValidateLimit(in.Limit, l.svcCtx.DefaultLimit(), l.svcCtx.MaxLimit())
	if err != nil {
		return nil, err
	}
	cursorAt, cursorID, err := DecodeCursor(in.Cursor)
	if err != nil {
		return nil, err
	}

	filter := operationsmodels.EventQueryFilter{
		MessageID:        in.MessageId,
		StartTime:        in.StartTime,
		EndTime:          in.EndTime,
		Limit:            int64(limit),
		HasCursor:        in.Cursor != "",
		CursorOccurredAt: cursorAt,
		CursorEventID:    cursorID,
	}

	events, err := l.svcCtx.EventModel.FindByMessageID(l.ctx, filter)
	if err != nil {
		return nil, types.MapQueryError(err)
	}

	observedAt := nowUnixNano()
	coverage, dropped, err := coverageFrom(l.ctx, l.svcCtx.EventModel, in.StartTime, in.EndTime, observedAt)
	if err != nil {
		return nil, err
	}

	out := make([]*operations.MessageEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, toProtoEvent(ev))
	}

	complete := int32(len(out)) < limit
	truncated := !complete
	// 有观测缺口时禁止 complete=true
	if coverage != types.CoverageComplete {
		complete = false
	}

	nextCursor := ""
	if truncated && len(events) > 0 {
		last := events[len(events)-1]
		nextCursor = EncodeCursor(last.OccurredAt, last.EventID)
	}

	return &operations.GetMessageTimelineResponse{
		Events:         out,
		Complete:       complete,
		Truncated:      truncated,
		NextCursor:     nextCursor,
		MessageId:      in.MessageId,
		CoverageStatus: coverage,
		EventsDropped:  dropped,
	}, nil
}

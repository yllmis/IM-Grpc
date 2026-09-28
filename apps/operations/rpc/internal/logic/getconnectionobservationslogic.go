package logic

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"github.com/IM_System/pkg/observation"
)

type GetConnectionObservationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConnectionObservationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConnectionObservationsLogic {
	return &GetConnectionObservationsLogic{ctx: ctx, svcCtx: svcCtx}
}

// GetConnectionObservations 区分「当前在线」与「历史当时是否在线」。
// 禁止用当前内存表冒充历史；覆盖不足必须如实标注。
func (l *GetConnectionObservationsLogic) GetConnectionObservations(in *operations.GetConnectionObservationsRequest) (*operations.GetConnectionObservationsResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if in.UserId == "" {
		return nil, types.InvalidArgument("userId is required")
	}
	// at 与时间范围互斥
	if in.At > 0 && (in.StartTime > 0 || in.EndTime > 0) {
		return nil, types.InvalidArgument("at is mutually exclusive with startTime/endTime")
	}
	if err := types.ValidateTimeRange(in.StartTime, in.EndTime); err != nil {
		return nil, err
	}
	if in.At < 0 {
		return nil, types.InvalidArgument("at must be UnixNano >= 0")
	}

	limit, err := types.ValidateLimit(in.Limit, l.svcCtx.DefaultLimit(), l.svcCtx.MaxLimit())
	if err != nil {
		return nil, err
	}
	cursorAt, cursorID, err := DecodeCursor(in.Cursor)
	if err != nil {
		return nil, err
	}

	start, end := in.StartTime, in.EndTime
	note := ""
	if in.At > 0 {
		// at 语义：取 observedAt <= at 的事件用于状态推演
		end = in.At
		note = "at-point-query"
	}

	filter := operationsmodels.EventQueryFilter{
		ReceiverID:       in.UserId,
		EventTypes:       []string{observation.EventOnline, observation.EventOffline},
		StartTime:        start,
		EndTime:          end,
		Limit:            int64(limit),
		HasCursor:        in.Cursor != "",
		CursorOccurredAt: cursorAt,
		CursorEventID:    cursorID,
	}

	events, err := l.svcCtx.EventModel.FindBySender(l.ctx, filter)
	if err != nil {
		return nil, types.MapQueryError(err)
	}

	observedAt := nowUnixNano()
	coverage, dropped, err := coverageFrom(l.ctx, l.svcCtx.EventModel, start, end, observedAt)
	if err != nil {
		return nil, err
	}

	obs := make([]*operations.ConnectionObservation, 0, len(events))
	for _, ev := range events {
		state := "offline"
		if ev.EventType == observation.EventOnline {
			state = "online"
		}
		reason := ""
		connectionID := ""
		instanceID := ""
		if ev.Metadata != nil {
			reason = ev.Metadata["reason"]
			connectionID = ev.Metadata["connectionId"]
			instanceID = ev.Metadata["instanceId"]
		}
		obs = append(obs, &operations.ConnectionObservation{
			ConnectionId: connectionID,
			InstanceId:   instanceID,
			State:        state,
			ObservedAt:   ev.OccurredAt,
			Reason:       reason,
		})
	}

	complete := int32(len(obs)) < limit
	truncated := !complete
	if coverage != types.CoverageComplete {
		complete = false
		if note == "" {
			note = "partial-history-before-rollout"
		}
	}

	nextCursor := ""
	if truncated && len(events) > 0 {
		last := events[len(events)-1]
		nextCursor = EncodeCursor(last.OccurredAt, last.EventID)
	}

	resp := &operations.GetConnectionObservationsResponse{
		Observations:   obs,
		Complete:       complete,
		Truncated:      truncated,
		NextCursor:     nextCursor,
		Note:           note,
		CoverageStatus: coverage,
		EventsDropped:  dropped,
	}

	// 当前状态：本查询服务不持有 WS 连接表，诚实标注 unknown 来源
	if in.IncludeCurrent {
		resp.Current = &operations.CurrentConnection{
			Online:     false,
			ObservedAt: observedAt,
			Source:     "unknown",
		}
		resp.Note = joinNote(resp.Note, "current-state-not-authoritative-in-operations-query")
	}

	return resp, nil
}

func joinNote(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "; " + b
}

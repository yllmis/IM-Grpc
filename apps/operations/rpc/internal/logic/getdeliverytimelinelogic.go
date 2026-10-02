package logic

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/faultinject"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/internal/types"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
)

type GetDeliveryTimelineLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDeliveryTimelineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeliveryTimelineLogic {
	return &GetDeliveryTimelineLogic{ctx: ctx, svcCtx: svcCtx}
}

// GetDeliveryTimeline 投递侧事件流；群聊按 receiverId 独立拆分。
// delivery_succeeded 的 evidence=gateway-write，禁止表述为已送达。
func (l *GetDeliveryTimelineLogic) GetDeliveryTimeline(in *operations.GetDeliveryTimelineRequest) (*operations.GetDeliveryTimelineResponse, error) {
	if in == nil {
		return nil, types.InvalidArgument("request is required")
	}
	if err := types.ValidateMessageID(in.MessageId); err != nil {
		return nil, err
	}
	if response, injected, err := l.injectedDeliveryTimeline(in.MessageId); injected {
		return response, err
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
		ReceiverID:       in.ReceiverId,
		EventTypes:       deliveryEventTypes(),
		Limit:            int64(limit),
		HasCursor:        in.Cursor != "",
		CursorOccurredAt: cursorAt,
		CursorEventID:    cursorID,
	}

	events, err := l.svcCtx.EventModel.FindByReceiver(l.ctx, filter)
	if err != nil {
		return nil, types.MapQueryError(err)
	}

	observedAt := nowUnixNano()
	coverage, dropped, err := coverageFrom(l.ctx, l.svcCtx.EventModel, 0, 0, observedAt)
	if err != nil {
		return nil, err
	}

	out := make([]*operations.DeliveryEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, &operations.DeliveryEvent{
			EventId:    ev.EventID,
			EventType:  ev.EventType,
			MessageId:  ev.MessageID,
			ReceiverId: ev.ReceiverID,
			AttemptId:  ev.AttemptID,
			OccurredAt: ev.OccurredAt,
			Source:     ev.Source,
			ErrorCode:  ev.ErrorCode,
			Metadata:   ev.Metadata,
			Evidence:   evidenceOf(ev.EventType),
		})
	}

	complete := int32(len(out)) < limit
	truncated := !complete
	if coverage != types.CoverageComplete {
		complete = false
	}

	nextCursor := ""
	if truncated && len(events) > 0 {
		last := events[len(events)-1]
		nextCursor = EncodeCursor(last.OccurredAt, last.EventID)
	}

	return &operations.GetDeliveryTimelineResponse{
		Events:         out,
		Complete:       complete,
		Truncated:      truncated,
		NextCursor:     nextCursor,
		MessageId:      in.MessageId,
		CoverageStatus: coverage,
		EventsDropped:  dropped,
	}, nil
}

// injectedDeliveryTimeline returns bounded synthetic events for one configured
// message ID. It does not touch MongoDB or the delivery pipeline.
func (l *GetDeliveryTimelineLogic) injectedDeliveryTimeline(messageID string) (*operations.GetDeliveryTimelineResponse, bool, error) {
	scenario, ok := l.svcCtx.FaultInjection.ScenarioFor(messageID)
	if !ok {
		return nil, false, nil
	}
	switch scenario {
	case faultinject.DeliveryEmpty:
		return &operations.GetDeliveryTimelineResponse{
			Events:         []*operations.DeliveryEvent{},
			Complete:       true,
			MessageId:      messageID,
			CoverageStatus: types.CoverageComplete,
		}, true, nil
	case faultinject.DeliveryTimeout:
		return nil, true, types.MapQueryError(context.DeadlineExceeded)
	case faultinject.UnsupportedCapability:
		return nil, true, types.Unimplemented("fault injection unsupported capability")
	case faultinject.ReceiverOffline, faultinject.AckTimeout:
		eventType := "receiver_offline"
		if scenario == faultinject.AckTimeout {
			eventType = "ack_timeout"
		}
		return &operations.GetDeliveryTimelineResponse{
			Events: []*operations.DeliveryEvent{{
				EventId:    "fault-" + string(scenario),
				EventType:  eventType,
				MessageId:  messageID,
				OccurredAt: nowUnixNano(),
				Source:     "fault-injection",
				ErrorCode:  string(scenario),
				Evidence:   "fault-injection-config",
			}},
			Complete:       true,
			MessageId:      messageID,
			CoverageStatus: types.CoverageComplete,
		}, true, nil
	default:
		return nil, false, nil
	}
}

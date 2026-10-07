package server

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/logic"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
)

// ObservationServer is the permanent observation-only contract. The legacy
// OperationsQuery facade remains until connectors finish migration.
type ObservationServer struct {
	operations.UnimplementedObservationQueryServer
	svcCtx *svc.ServiceContext
}

func NewObservationServer(s *svc.ServiceContext) *ObservationServer {
	return &ObservationServer{svcCtx: s}
}
func (s *ObservationServer) GetMessageTimeline(ctx context.Context, in *operations.GetMessageTimelineRequest) (*operations.GetMessageTimelineResponse, error) {
	return logic.NewGetMessageTimelineLogic(ctx, s.svcCtx).GetMessageTimeline(in)
}
func (s *ObservationServer) GetDeliveryTimeline(ctx context.Context, in *operations.GetDeliveryTimelineRequest) (*operations.GetDeliveryTimelineResponse, error) {
	return logic.NewGetDeliveryTimelineLogic(ctx, s.svcCtx).GetDeliveryTimeline(in)
}
func (s *ObservationServer) GetConnectionObservations(ctx context.Context, in *operations.GetConnectionObservationsRequest) (*operations.GetConnectionObservationsResponse, error) {
	return logic.NewGetConnectionObservationsLogic(ctx, s.svcCtx).GetConnectionObservations(in)
}
func (s *ObservationServer) GetCapabilities(ctx context.Context, in *operations.GetCapabilitiesRequest) (*operations.GetCapabilitiesResponse, error) {
	resp, err := logic.NewGetCapabilitiesLogic(ctx, s.svcCtx).GetCapabilities(in)
	if err != nil {
		return nil, err
	}
	// Capability declarations describe this contract, never another service.
	resp.MessageRecord = "unsupported"
	resp.MessageSearch = "unsupported"
	return resp, nil
}

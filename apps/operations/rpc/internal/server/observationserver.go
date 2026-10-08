package server

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/logic"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
)

// ObservationServer 只持有观测数据，不转调 User/IM 的业务查询。
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
	return logic.NewGetCapabilitiesLogic(ctx, s.svcCtx).GetCapabilities(in)
}

package server

import (
	"context"

	"github.com/IM_System/apps/operations/rpc/internal/logic"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
)

type OperationsServer struct {
	svcCtx *svc.ServiceContext
	operations.UnimplementedOperationsQueryServer
}

func NewOperationsServer(svcCtx *svc.ServiceContext) *OperationsServer {
	return &OperationsServer{svcCtx: svcCtx}
}

func (s *OperationsServer) FindUserReference(ctx context.Context, in *operations.FindUserReferenceRequest) (*operations.FindUserReferenceResponse, error) {
	return logic.NewFindUserReferenceLogic(ctx, s.svcCtx).FindUserReference(in)
}

func (s *OperationsServer) GetMessageRecord(ctx context.Context, in *operations.GetMessageRecordRequest) (*operations.GetMessageRecordResponse, error) {
	return logic.NewGetMessageRecordLogic(ctx, s.svcCtx).GetMessageRecord(in)
}

func (s *OperationsServer) GetMessageTimeline(ctx context.Context, in *operations.GetMessageTimelineRequest) (*operations.GetMessageTimelineResponse, error) {
	return logic.NewGetMessageTimelineLogic(ctx, s.svcCtx).GetMessageTimeline(in)
}

func (s *OperationsServer) GetDeliveryTimeline(ctx context.Context, in *operations.GetDeliveryTimelineRequest) (*operations.GetDeliveryTimelineResponse, error) {
	return logic.NewGetDeliveryTimelineLogic(ctx, s.svcCtx).GetDeliveryTimeline(in)
}

func (s *OperationsServer) GetConnectionObservations(ctx context.Context, in *operations.GetConnectionObservationsRequest) (*operations.GetConnectionObservationsResponse, error) {
	return logic.NewGetConnectionObservationsLogic(ctx, s.svcCtx).GetConnectionObservations(in)
}

func (s *OperationsServer) GetCapabilities(ctx context.Context, in *operations.GetCapabilitiesRequest) (*operations.GetCapabilitiesResponse, error) {
	return logic.NewGetCapabilitiesLogic(ctx, s.svcCtx).GetCapabilities(in)
}

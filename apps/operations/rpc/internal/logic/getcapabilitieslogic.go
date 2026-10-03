package logic

import (
	"context"
	"strconv"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/observation"
)

type GetCapabilitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCapabilitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCapabilitiesLogic {
	return &GetCapabilitiesLogic{ctx: ctx, svcCtx: svcCtx}
}

// GetCapabilities 诚实反映本实例实际配置，禁止虚报 supported。
// 取值仅 supported | partial | unsupported。
func (l *GetCapabilitiesLogic) GetCapabilities(_ *operations.GetCapabilitiesRequest) (*operations.GetCapabilitiesResponse, error) {
	enabled := l.svcCtx.Config.DeliveryObservation.Enabled
	ackMode := l.svcCtx.Config.DeliveryObservation.AckMode
	if ackMode == "" {
		ackMode = observation.AckModeNoAck
	}

	// message_record：不依赖观测开关
	messageRecord := "supported"

	// 其余能力依赖观测事件；关闭时一律 unsupported
	messageTimeline := "unsupported"
	deliveryEvents := "unsupported"
	historicalConnection := "unsupported"
	writeFailureEvents := "unsupported"
	readConfirmation := "unsupported"
	ackHistory := "unsupported"
	messageSearch := "unsupported"
	if l.svcCtx.MessageSearch != nil {
		messageSearch = "supported"
	}

	if enabled {
		// 观测开启但可能丢事件 → partial（覆盖可证明完整前不上调 supported）
		messageTimeline = "partial"
		deliveryEvents = "partial"
		writeFailureEvents = "partial"
		readConfirmation = "partial"

		// 连接事件与 ACK 依赖额外条件
		historicalConnection = "partial"
		if ackMode == observation.AckModeRigorAck {
			ackHistory = "partial"
		}
		// NoAck/OnlyAck：不产生真实客户端 ACK，保持 unsupported
	}

	return &operations.GetCapabilitiesResponse{
		MessageRecord:        messageRecord,
		MessageSearch:        messageSearch,
		MessageTimeline:      messageTimeline,
		DeliveryEvents:       deliveryEvents,
		HistoricalConnection: historicalConnection,
		AckHistory:           ackHistory,
		WriteFailureEvents:   writeFailureEvents,
		ReadConfirmation:     readConfirmation,
		ObservedAt:           nowUnixNano(),
		ObservationEnabled:   strconv.FormatBool(enabled),
		AckMode:              ackMode,
	}, nil
}

package push

import (
	"context"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/observation"
	"github.com/mitchellh/mapstructure"
)

// 用于消息的转发和处理
// 投递事件为旁路观测：业务返回值与旧版保持一致（离线仍视为成功）。

func Push(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Push
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}

		switch data.ChatType {
		case constants.SingleChatType:
			single(srv, svc, &data, data.RecvId)

		case constants.GroupChatType:
			group(srv, svc, &data)
		}
	}
}

func single(srv *websocket.Server, svc *svc.ServiceContext, data *ws.Push, RecvId string) error {
	builder := observation.MessageEventBuilder{
		MessageID:      data.MsgId,
		ConversationID: data.ConversationId,
		SenderID:       data.SendId,
		ReceiverID:     RecvId,
		Source:         observation.SourceWsPush,
	}

	// 发送的目标
	rconn := srv.GetConn(RecvId)
	if rconn == nil {
		observation.SafeRecord(context.Background(), svc.ObservationSink, builder.ReceiverOffline())
		return nil
	}

	observation.SafeRecord(context.Background(), svc.ObservationSink, builder.DeliveryAttempted())

	srv.Infof("push msg: %v", data)

	err := srv.Send(websocket.NewMessage(data.SendId, ws.Chat{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendTime:       data.SendTime,
		Msg: ws.Msg{
			ReadRecords: data.ReadRecords,
			MsgId:       data.MsgId,
			MType:       data.MType,
			Content:     data.Content,
		},
	}), rconn)

	if err != nil {
		observation.SafeRecord(context.Background(), svc.ObservationSink, builder.DeliveryFailed(observation.ErrCodeDeliveryWriteFailed))
		return err
	}

	// delivery_succeeded == 网关 socket 写出成功，不代表客户端已收到
	observation.SafeRecord(context.Background(), svc.ObservationSink, builder.DeliverySucceeded())
	return err
}

func group(srv *websocket.Server, svc *svc.ServiceContext, data *ws.Push) error {
	// 群聊：每个接收方独立投递记录
	for _, id := range data.RecvIds {
		func(id string) {
			srv.Schedule(func() {
				single(srv, svc, data, id)
			})
		}(id)
	}
	return nil
}

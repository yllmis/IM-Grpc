package push

import (
	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/pkg/constants"
	"github.com/mitchellh/mapstructure"
)

// 用于消息的转发和处理

func Push(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Push
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}

		switch data.ChatType {
		case constants.SingleChatType:
			single(srv, &data, data.RecvId)

		case constants.GroupChatType:
			group(srv, &data)
		}
	}
}

func single(srv *websocket.Server, data *ws.Push, RecvId string) error {
	// 发送的目标
	rconn := srv.GetConn(RecvId)
	if rconn == nil {
		return nil
	}

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
	return err
}

func group(srv *websocket.Server, data *ws.Push) error {
	for _, id := range data.RecvIds {
		func(id string) {
			srv.Schedule(func() {
				single(srv, data, id)
			})
		}(id)
	}
	return nil
}

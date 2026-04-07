package conversation

import (
	"context"
	"time"

	"github.com/IM_System/apps/im/ws/internal/logic"
	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/pkg/constants"
	"github.com/mitchellh/mapstructure"
)

func Chat(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		// todo: 处理私聊
		var data ws.Chat
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}
		switch data.ChatType {
		case constants.SingleChatType:
			// 处理单聊
			err := logic.NewConversation(context.Background(), srv, svc).SingleConversation(&data, conn.Uid)
			if err != nil {
				srv.Send(websocket.NewErrMessgae(err), conn)
			}

			srv.SendByUserId(websocket.NewMessage(conn.Uid, ws.Chat{
				ConversationId: data.ConversationId,
				SendId:         conn.Uid,
				RecvId:         data.RecvId,
				ChatType:       data.ChatType,
				SendTime:       time.Now().UnixNano(),
				Msg:            data.Msg,
			}), data.RecvId)
		}

	}

}

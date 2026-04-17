package conversation

import (
	"time"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/wuid"
	"github.com/mitchellh/mapstructure"
)

func Chat(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Chat
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}

		if data.ConversationId == "" {
			switch data.ChatType {
			case constants.SingleChatType:
				data.ConversationId = wuid.CombineId(conn.Uid, data.RecvId)
			case constants.GroupChatType:
				data.ConversationId = data.RecvId // 群聊的conversationId就是群id,RecvId 是群id
			}
		}

		err := svc.MsgChatTransferClient.Push(&mq.MsgChatTransfer{
			ConversationId: data.ConversationId,
			SendId:         conn.Uid,
			RecvId:         data.RecvId,
			ChatType:       data.ChatType,
			SendTime:       time.Now().UnixNano(),
			MType:          data.MType,
			Content:        data.Content,
		}) // 推送消息到消息队列中，等待后续处理

		if err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
		}
	}

}

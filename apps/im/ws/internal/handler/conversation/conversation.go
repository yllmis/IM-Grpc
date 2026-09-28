package conversation

import (
	"context"
	"time"

	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/observation"
	"github.com/IM_System/pkg/wuid"
	"github.com/mitchellh/mapstructure"
	"github.com/zeromicro/go-zero/core/logx"
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

		// 稳定消息 ID：仅在显式开启后生成并进入链路，保证默认零行为变化
		ident := NewMessageIdentity(svc.Config.MessageIdentity.EmitServerMessageId, msg.Id)

		// accepted：消息被接入层接受（旁路；无稳定 messageId 时不写消息事件）
		if ident.MessageId != "" {
			observation.SafeRecord(context.Background(), svc.ObservationSink, observation.MessageEventBuilder{
				MessageID:       ident.MessageId,
				ClientMessageID: ident.ClientMessageId,
				CorrelationID:   ident.CorrelationId,
				ConversationID:  data.ConversationId,
				SenderID:        conn.Uid,
				ReceiverID:      data.RecvId,
				Source:          observation.SourceImWs,
			}.Accepted())
		}

		err := svc.MsgChatTransferClient.Push(&mq.MsgChatTransfer{
			MessageId:       ident.MessageId,
			ClientMessageId: ident.ClientMessageId,
			CorrelationId:   ident.CorrelationId,
			ConversationId:  data.ConversationId,
			SendId:          conn.Uid,
			RecvId:          data.RecvId,
			ChatType:        data.ChatType,
			SendTime:        time.Now().UnixNano(),
			MType:           data.MType,
			Content:         data.Content,
		}) // 推送消息到消息队列中，等待后续处理

		if err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}

		// 业务ACK：消息已成功进入处理链路
		// msgId 保留为客户端请求 ID；serverMessageId 为服务端稳定消息 ID（可选新增字段）
		srv.Send(websocket.NewMessage(conn.Uid, SentResponse(ident, msg.Id)), conn)
	}

}

func MarkRead(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		// 已读未读处理
		logx.Infof("【收到已读请求】: %+v", msg)
		var data ws.MarkRead
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			logx.Errorf("解析已读请求参数失败: %v", err)
			srv.Send(websocket.NewErrMessgae(err), conn)
			return
		}

		if data.ConversationId == "" {
			switch data.ChatType {
			case constants.SingleChatType:
				data.ConversationId = wuid.CombineId(conn.Uid, data.RecvId)
			case constants.GroupChatType:
				data.ConversationId = data.RecvId // 群聊的conversationId就是群id, RecvId是群id
			}
		}

		err := svc.MsgReadTransferClient.Push(&mq.MsgMarkRead{
			ConversationId: data.ConversationId,
			SendId:         conn.Uid,
			RecvId:         data.RecvId,
			ChatType:       data.ChatType,
			MsgIds:         data.MsgIds,
		}) // 推送消息到消息队列中，等待后续处理

		if err != nil {
			srv.Send(websocket.NewErrMessgae(err), conn)
			logx.Errorf("推送已读消息到Kafka失败: %v", err)
		}
		logx.Info("推送已读消息到Kafka成功！")
	}

}

package msgtransfer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/bitmap"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MsgChatTransfer struct {
	*baseMsgTransfer
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{NewBaseMsgTransfer(svc)}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error { // 消费者，有更新消息时会调用这个方法
	fmt.Printf("consume msg, key: %s, value: %s\n", key, value)

	var (
		data  mq.MsgChatTransfer
		msgId = bson.NewObjectID()
	)

	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	// 记录数据
	if err := m.addChatLog(ctx, msgId, data); err != nil {
		return err
	}

	return m.Transfer(ctx, &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		RecvIds:        data.RecvIds,
		MsgId:          msgId.Hex(),
		SendTime:       data.SendTime,
		MType:          data.MType,
		Content:        data.Content,
	})
}

func (m *MsgChatTransfer) addChatLog(ctx context.Context, msgId bson.ObjectID, data mq.MsgChatTransfer) error {
	chatLog := &immodels.ChatLog{
		ID:             msgId,
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       data.SendTime,
	}

	readRecords := bitmap.NewBitmap(0)
	readRecords.Set(chatLog.SendId) // 发送者默认已读
	chatLog.ReadRecords = readRecords.Export()

	err := m.svcCtx.ChatLogModel.Insert(ctx, chatLog)
	if err != nil {
		return err
	}
	return m.svcCtx.ConversationModel.UpdateMsg(ctx, chatLog)
}

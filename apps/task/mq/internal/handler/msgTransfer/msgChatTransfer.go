package msgtransfer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-zero/core/logx"
)

type MsgChatTransfer struct {
	logx.Logger
	svc *svc.ServiceContext
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{
		Logger: logx.WithContext(context.Background()),
		svc:    svc,
	}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error { // 消费者，有更新消息时会调用这个方法
	fmt.Printf("consume msg, key: %s, value: %s\n", key, value)

	var (
		data mq.MsgChatTransfer
	)

	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	// 记录数据
	if err := m.addChatLog(ctx, data); err != nil {
		return err
	}

	switch data.ChatType {
	case constants.SingleChatType:
		return m.single(&data)
	case constants.GroupChatType:
		return m.group(ctx, &data)
	}
	return nil
}

func (m *MsgChatTransfer) single(data *mq.MsgChatTransfer) error {
	// 私聊推送
	// 推送消息
	return m.svc.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_ID,
		Data:      data,
	})
}

func (m *MsgChatTransfer) group(ctx context.Context, data *mq.MsgChatTransfer) error {
	// 群聊推送
	// 获取群成员列表
	users, err := m.svc.Social.Groupusers(ctx, &social.GroupusersReq{
		GroupId: data.RecvId,
	})
	if err != nil {
		return err
	}

	data.RecvIds = make([]string, 0, len((users.List)))
	for _, members := range users.List {
		if members.UserId == data.SendId {
			continue
		}
		data.RecvIds = append(data.RecvIds, members.UserId)
	}

	return m.svc.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_ID,
		Data:      data,
	})
}

func (m *MsgChatTransfer) addChatLog(ctx context.Context, data mq.MsgChatTransfer) error {
	chatLog := &immodels.ChatLog{
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       data.SendTime,
	}
	err := m.svc.ChatLogModel.Insert(ctx, chatLog)
	if err != nil {
		return err
	}
	return m.svc.ConversationModel.UpdateMsg(ctx, chatLog)
}

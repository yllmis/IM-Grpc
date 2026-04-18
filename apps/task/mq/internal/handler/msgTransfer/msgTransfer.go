package msgtransfer

import (
	"context"

	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-zero/core/logx"
)

type baseMsgTransfer struct {
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBaseMsgTransfer(svcCtx *svc.ServiceContext) *baseMsgTransfer {
	return &baseMsgTransfer{
		svcCtx: svcCtx,
		Logger: logx.WithContext(context.Background()),
	}
}

func (m *baseMsgTransfer) Transfer(ctx context.Context, data *ws.Push) error {
	switch data.ChatType {
	case constants.SingleChatType:
		return m.single(ctx, data)
	case constants.GroupChatType:
		return m.group(ctx, data)
	}

	return nil
}

func (m *baseMsgTransfer) single(ctx context.Context, data *ws.Push) error {
	// 私聊推送
	// 推送消息
	return m.svcCtx.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_ID,
		Data:      data,
	})
}

func (m *baseMsgTransfer) group(ctx context.Context, data *ws.Push) error {
	// 群聊推送
	// 获取群成员列表
	users, err := m.svcCtx.Social.Groupusers(ctx, &social.GroupusersReq{
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

	return m.svcCtx.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameData,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_ID,
		Data:      data,
	})
}

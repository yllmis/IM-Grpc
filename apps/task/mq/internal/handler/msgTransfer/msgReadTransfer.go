package msgtransfer

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/bitmap"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-queue/kq"
)

// 处理已读和未读

type MsgReadTransfer struct {
	*baseMsgTransfer
}

func NewMsgReadTransfer(svcCtx *svc.ServiceContext) kq.ConsumeHandler {
	return &MsgReadTransfer{
		NewBaseMsgTransfer(svcCtx),
	}
}

func (m *MsgReadTransfer) Consume(ctx context.Context, key, value string) error {
	m.Infof("MsgReadTransfer: %s", value)

	var (
		data mq.MsgMarkRead
	)
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	// 业务处理 -- 更新

	res, err := m.UpdateChatLogRead(ctx, &data)
	if err != nil {
		return err
	}

	return m.Transfer(ctx, &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ContentType:    constants.ContentReadType,
		ReadRecords:    res,
	})
}

func (m *MsgReadTransfer) UpdateChatLogRead(ctx context.Context, data *mq.MsgMarkRead) (map[string]string, error) {
	// 更新聊天记录的已读状态
	// 1. 获取聊天记录
	// 2. 更新已读状态
	// 3. 返回更新结果
	res := make(map[string]string)

	chatlogs, err := m.svcCtx.ChatLogModel.ListByIds(ctx, data.MsgIds)
	if err != nil {
		return nil, err
	}

	// 处理已读
	for _, chatlog := range chatlogs {
		switch chatlog.ChatType {
		case constants.SingleChatType:
			// 私聊，直接更新
			chatlog.ReadRecords = []byte{1} // 已读
		case constants.GroupChatType:
			// 群聊，更新对应用户的已读状态
			readRecords := bitmap.Load(chatlog.ReadRecords)
			readRecords.Set(data.SendId) // 标记为已读
			chatlog.ReadRecords = readRecords.Export()
		}

		res[chatlog.ID.Hex()] = base64.StdEncoding.EncodeToString(chatlog.ReadRecords)

		err := m.svcCtx.ChatLogModel.UpdateMakeRead(ctx, chatlog.ID, chatlog.ReadRecords)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

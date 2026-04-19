package msgtransfer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/bitmap"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/cache"
)

var (
	GroupMsgReadRecordDelayTime  = time.Second
	GroupMsgReadRecordDelayCount = 10
)

const (
	GroupMsgReadHandlerAtTransfer = iota
	GroupMsgReadHandlerDelayTransfer
)

// 处理已读和未读
type MsgReadTransfer struct {
	*baseMsgTransfer

	cache.Cache

	mu sync.Mutex

	groupMsgs map[string]*groupMsgRead
	push      chan *ws.Push
}

func NewMsgReadTransfer(svcCtx *svc.ServiceContext) kq.ConsumeHandler {
	m := &MsgReadTransfer{
		baseMsgTransfer: NewBaseMsgTransfer(svcCtx),
		groupMsgs:       make(map[string]*groupMsgRead),
		push:            make(chan *ws.Push, 1),
	}

	if svcCtx.Config.MsgReadHandler.GroupMsgReadRecordDelayTime > 0 {
		GroupMsgReadRecordDelayTime = time.Duration(svcCtx.Config.MsgReadHandler.GroupMsgReadRecordDelayTime) * time.Second
	}
	if svcCtx.Config.MsgReadHandler.GroupMsgReadRecordDelayCount > 0 {
		GroupMsgReadRecordDelayCount = svcCtx.Config.MsgReadHandler.GroupMsgReadRecordDelayCount
	}

	go m.transfer()

	return m
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

	push := &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ContentType:    constants.ContentReadType,
		ReadRecords:    res,
	}

	switch data.ChatType {
	case constants.SingleChatType:
		// 私聊，直接推送
		m.push <- push
	case constants.GroupChatType:
		// 判断是否开启合并消息的处理
		if m.svcCtx.Config.MsgReadHandler.GroupMsgReadHandler == GroupMsgReadHandlerAtTransfer {
			m.push <- push
			return nil
		}

		m.mu.Lock()
		defer m.mu.Unlock()
		push.SendId = "" // 群聊不区分发送者，合并消息时会有多个发送者，所以不区分发送者

		if _, ok := m.groupMsgs[push.ConversationId]; ok {
			// 已经有未推送的消息了，合并消息
			m.Infof("merge push %v", push.ConversationId)
			m.groupMsgs[push.ConversationId].mergePush(push)
		} else {
			m.Infof("newGroupMsgRead push %v", push.ConversationId)
			m.groupMsgs[push.ConversationId] = newGroupMsgRead(push, m.push)
		}
	}

	return nil

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

func (m *MsgReadTransfer) transfer() {
	for push := range m.push {
		if push.RecvId != "" || len(push.RecvIds) > 0 {
			if err := m.Transfer(context.Background(), push); err != nil {
				m.Errorf("m transfer err %v push %v", err, push)
			}
		}

		if push.ChatType == constants.SingleChatType {
			continue
		}

		// 清空数据
		m.mu.Lock()
		if _, ok := m.groupMsgs[push.ConversationId]; ok && m.groupMsgs[push.ConversationId].IsIdle() { // 如果闲置了，就删除
			m.groupMsgs[push.ConversationId].clear()
			delete(m.groupMsgs, push.ConversationId)
		}
		m.mu.Unlock()
	}
}

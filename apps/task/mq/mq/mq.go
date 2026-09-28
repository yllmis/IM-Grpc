package mq

import "github.com/IM_System/pkg/constants"

type MsgChatTransfer struct {
	// 可选关联 ID；旧生产者不带这些字段，消费者必须兼容空值
	MessageId       string `json:"messageId,omitempty"`       // 服务端稳定消息 ID（24hex ObjectID）
	ClientMessageId string `json:"clientMessageId,omitempty"` // 客户端原始 msg.Id
	CorrelationId   string `json:"correlationId,omitempty"`   // 链路追踪 ID

	ConversationId     string `json:"conversationId"`
	constants.ChatType `json:"chatType"`
	SendId             string   `json:"sendId"`
	RecvId             string   `json:"recvId"`
	RecvIds            []string `json:"recvIds"` // 群聊时使用
	SendTime           int64    `json:"sendTime"`

	constants.MType `json:"mType"`
	Content         string `json:"content"`
}

type MsgMarkRead struct {
	constants.ChatType `json:"chatType"`
	ConversationId     string   `json:"conversationId"`
	SendId             string   `json:"sendId"`
	RecvId             string   `json:"recvId"`
	MsgIds             []string `json:"msgIds"` // 需要标记为已读的消息ID列表
}

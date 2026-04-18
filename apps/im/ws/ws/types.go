package ws

import "github.com/IM_System/pkg/constants"

type (
	Msg struct {
		constants.MType `mapstructure:"mType"`
		Content         string            `mapstructure:"content"`
		MsgId           string            `mapstructure:"msgId"`
		ReadRecords     map[string]string `mapstructure:"readRecords"`
	}

	Chat struct {
		ConversationId     string `mapstructure:"conversationId"`
		constants.ChatType `mapstructure:"chatType"`
		SendId             string `mapstructure:"sendId"`
		RecvId             string `mapstructure:"recvId"`
		Msg                `mapstructure:"msg"`
		SendTime           int64 `mapstructure:"sendTime"`
	}

	Push struct {
		ConversationId     string `mapstructure:"conversationId"`
		constants.ChatType `mapstructure:"chatType"`
		SendId             string   `mapstructure:"sendId"`
		RecvId             string   `mapstructure:"recvId"`
		RecvIds            []string `mapstructure:"recvIds"`
		SendTime           int64    `mapstructure:"sendTime"`

		MsgId string `mapstructure:"msgId"`

		ReadRecords map[string]string     `mapstructure:"readRecords"` // 消息ID -> 已读用户ID列表，base64编码的bitmap
		ContentType constants.ContentType `mapstructure:"contentType"`

		constants.MType `mapstructure:"mType"`
		Content         string `mapstructure:"content"`
	}

	MarkRead struct {
		ConversationId     string `mapstructure:"conversationId"`
		constants.ChatType `mapstructure:"chatType"`
		RecvId             string   `mapstructure:"recvId"`
		MsgIds             []string `mapstructure:"msgIds"`
	}
)

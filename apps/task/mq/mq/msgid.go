package mq

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResolveMessageId 决定消费侧使用的稳定消息 ID。
// 新消息（MessageId 为 24hex ObjectID）直接沿用，禁止重新生成；
// 旧消息（无 MessageId）按旧行为生成 ObjectID，并由调用方标注 compat-legacy。
func ResolveMessageId(msg *MsgChatTransfer) (id bson.ObjectID, legacy bool) {
	if msg == nil {
		return bson.NewObjectID(), true
	}

	trimmed := strings.TrimSpace(msg.MessageId)
	if trimmed == "" {
		return bson.NewObjectID(), true
	}

	oid, err := bson.ObjectIDFromHex(trimmed)
	if err != nil {
		// 形态非法视作旧消息，保证消费不中断
		return bson.NewObjectID(), true
	}
	return oid, false
}

// NormalizeCorrelationId 返回链路追踪 ID；缺省等于 messageId。
func NormalizeCorrelationId(msg *MsgChatTransfer, messageIdHex string) string {
	if msg == nil {
		return messageIdHex
	}
	if strings.TrimSpace(msg.CorrelationId) != "" {
		return msg.CorrelationId
	}
	return messageIdHex
}

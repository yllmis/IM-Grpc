package conversation

import "go.mongodb.org/mongo-driver/v2/bson"

// MessageIdentity 承载稳定消息 ID 贯通所需字段。
type MessageIdentity struct {
	MessageId       string
	ClientMessageId string
	CorrelationId   string
}

// NewMessageIdentity 仅在 emit=true 时生成稳定 messageId。
// emit=false 保持旧链路零行为变化（不生成、不进 Kafka、回包无 serverMessageId）。
func NewMessageIdentity(emit bool, clientMsgId string) MessageIdentity {
	if !emit {
		return MessageIdentity{}
	}
	id := bson.NewObjectID().Hex()
	return MessageIdentity{
		MessageId:       id,
		ClientMessageId: clientMsgId,
		CorrelationId:   id,
	}
}

// SentResponse 构造业务回包。
// 始终保留 msgId + status:"sent"；仅在开启稳定 ID 时追加 serverMessageId。
func SentResponse(ident MessageIdentity, clientMsgId string) map[string]any {
	resp := map[string]any{
		"msgId":  clientMsgId,
		"status": "sent",
	}
	if ident.MessageId != "" {
		resp["serverMessageId"] = ident.MessageId
	}
	return resp
}

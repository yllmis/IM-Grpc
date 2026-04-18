package immodels

import (
	"time"

	"github.com/IM_System/pkg/constants"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ChatLog struct {
	ID bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`

	ConversationId string             `bson:"conversationId"`
	SendId         string             `bson:"sendId"`
	RecvId         string             `bson:"recvId"`
	ChatType       constants.ChatType `bson:"chatType"`
	MsgFrom        int                `bson:"msgFrom"`
	MsgType        constants.MType    `bson:"msgType"`
	MsgContent     string             `bson:"msgContent"`
	SendTime       int64              `bson:"sendTime"`
	Status         int                `bson:"status"`
	ReadRecords    []byte             `bson:"readRecords"` // 已读记录，使用bitmap存储，记录已读用户ID

	// TODO: Fill your own fields
	UpdateAt time.Time `bson:"updateAt,omitempty" json:"updateAt,omitempty"`
	CreateAt time.Time `bson:"createAt,omitempty" json:"createAt,omitempty"`
}

package immodels

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ ConversationModel = (*customConversationModel)(nil)

type (
	// ConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationModel.
	ConversationModel interface {
		conversationModel
	}

	customConversationModel struct {
		*defaultConversationModel
	}
)

// NewConversationModel returns a model for the mongo.
func NewConversationModel(url, db, collection string) ConversationModel {
	conn := mon.MustNewModel(url, db, collection)
	return &customConversationModel{
		defaultConversationModel: newDefaultConversationModel(conn),
	}
}

func MustConversationModel(url, db string) ConversationModel {
	return NewConversationModel(url, db, "conversation")
}

// UpdateMsg increments the count and conditionally replaces the summary in one
// atomic update. Concurrent processors must not overwrite a newer message with
// an older one. MongoDB 4.2+ supports this aggregation update pipeline.
func (m *customConversationModel) UpdateMsg(ctx context.Context, chatLog *ChatLog) error {
	_, err := m.conn.UpdateOne(ctx,
		bson.M{"conversationId": chatLog.ConversationId},
		bson.A{bson.M{"$set": bson.M{
			"total": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$total", 0}}, 1}},
			"msg": bson.M{"$cond": bson.A{
				bson.M{"$or": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$type": "$msg.sendTime"}, "missing"}},
					bson.M{"$lte": bson.A{"$msg.sendTime", bson.M{"$literal": chatLog.SendTime}}},
				}},
				bson.M{"$literal": chatLog},
				"$msg",
			}},
		}}})
	return err
}

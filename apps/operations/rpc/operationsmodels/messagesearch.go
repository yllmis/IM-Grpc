package operationsmodels

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MessageSearchModel 只读投影，不复用带写方法的业务 Model；索引由运维显式创建。
type MessageSearchModel interface {
	Search(context.Context, MessageSearchFilter) ([]MessageReference, error)
}
type MessageSearchFilter struct {
	SenderID, ReceiverID string
	StartTime, EndTime   int64
	Limit                int32
}
type MessageReference struct {
	ID             bson.ObjectID `bson:"_id"`
	ConversationID string        `bson:"conversationId"`
	SenderID       string        `bson:"sendId"`
	ReceiverID     string        `bson:"recvId"`
	CreatedAt      int64         `bson:"sendTime"`
}
type messageSearchModel struct{ coll *mongo.Collection }

func MustMessageSearchModel(url, db string) MessageSearchModel {
	client, err := mongo.Connect(options.Client().ApplyURI(url))
	if err != nil {
		panic(fmt.Errorf("message search connect: %w", err))
	}
	return &messageSearchModel{coll: client.Database(db).Collection("chat_log")}
}
func messageSearchQuery(f MessageSearchFilter) bson.M {
	query := bson.M{"sendId": f.SenderID, "sendTime": bson.M{"$gte": f.StartTime, "$lte": f.EndTime}}
	if f.ReceiverID != "" {
		query["recvId"] = f.ReceiverID
	}
	return query
}
func (m *messageSearchModel) Search(ctx context.Context, f MessageSearchFilter) ([]MessageReference, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()
	// 多读取一条仅用于判断截断；正文、密码、已读 bitmap 不进入返回结构。
	cursor, err := m.coll.Find(ctx, messageSearchQuery(f), options.Find().
		SetProjection(bson.M{"_id": 1, "conversationId": 1, "sendId": 1, "recvId": 1, "sendTime": 1}).
		SetSort(bson.D{{Key: "sendTime", Value: 1}, {Key: "_id", Value: 1}}).
		SetLimit(int64(f.Limit)+1))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []MessageReference
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

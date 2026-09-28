package observation

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CollectionMessageEvents 事件集合名（与 chat_log 同库）。
const CollectionMessageEvents = "message_events"

// MongoObservationSink 将事件写入 message_events。
// 幂等：以 eventId 为唯一键，重复写入不产生第二条记录。
type MongoObservationSink struct {
	coll *mongo.Collection
}

// NewMongoObservationSink 连接 Mongo 并确保索引存在。
func NewMongoObservationSink(ctx context.Context, url, db string) (*MongoObservationSink, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(url))
	if err != nil {
		return nil, fmt.Errorf("observation: connect mongo: %w", err)
	}

	coll := client.Database(db).Collection(CollectionMessageEvents)
	sink := &MongoObservationSink{coll: coll}
	if err := sink.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return sink, nil
}

// EnsureIndexes 创建契约要求的索引。可重复调用。
func (s *MongoObservationSink) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "eventId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_event_id"),
		},
		{
			Keys:    bson.D{{Key: "messageId", Value: 1}, {Key: "occurredAt", Value: 1}},
			Options: options.Index().SetName("idx_message_occurred"),
		},
		{
			Keys:    bson.D{{Key: "correlationId", Value: 1}},
			Options: options.Index().SetName("idx_correlation"),
		},
		{
			Keys:    bson.D{{Key: "receiverId", Value: 1}, {Key: "occurredAt", Value: 1}},
			Options: options.Index().SetName("idx_receiver_occurred"),
		},
		{
			Keys:    bson.D{{Key: "eventType", Value: 1}, {Key: "occurredAt", Value: 1}},
			Options: options.Index().SetName("idx_event_type_occurred"),
		},
	}

	_, err := s.coll.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("observation: ensure indexes: %w", err)
	}
	return nil
}

// Record 幂等写入单条事件。重复 eventId 不报错、不产生副本。
func (s *MongoObservationSink) Record(ctx context.Context, event MessageEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	if event.EventVersion == 0 {
		event.EventVersion = EventVersion
	}

	filter := bson.M{"eventId": event.EventID}
	update := bson.M{"$setOnInsert": event}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := s.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		// 唯一键冲突视为已存在，幂等成功
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("observation: record event: %w", err)
	}
	return nil
}

// NewObservationGap 构造观测缺口标记事件。
// metadata 至少包含 droppedCount / gapStartAt / gapEndAt / reason。
func NewObservationGap(source string, dropped int64, gapStartAt, gapEndAt int64, reason string) MessageEvent {
	if gapEndAt < gapStartAt {
		gapEndAt = gapStartAt
	}
	return MessageEvent{
		EventID:      NewEventID(),
		EventVersion: EventVersion,
		EventType:    EventObservationGap,
		Source:       source,
		OccurredAt:   gapEndAt,
		Metadata: map[string]string{
			"droppedCount": fmt.Sprintf("%d", dropped),
			"gapStartAt":   fmt.Sprintf("%d", gapStartAt),
			"gapEndAt":     fmt.Sprintf("%d", gapEndAt),
			"reason":       reason,
		},
	}
}

// NowUnixNano 当前时间 UnixNano。统一时间入口，避免混用单位。
func NowUnixNano() int64 { return time.Now().UnixNano() }

package operationsmodels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IM_System/pkg/observation"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// defaultQueryTimeout 单次查询预算；超时映射 DEADLINE_EXCEEDED，不得当成“无数据”。
const defaultQueryTimeout = 3 * time.Second

// ErrNotFound 查询成功但无记录（不是错误语义上的失败）。
var ErrNotFound = errors.New("operationsmodels: not found")

// EventQueryFilter 时间线查询条件。
type EventQueryFilter struct {
	MessageID  string
	ReceiverID string
	// EventTypes 为空表示不过滤类型
	EventTypes []string
	StartTime  int64 // UnixNano，0=不限
	EndTime    int64 // UnixNano，0=不限
	Limit      int64
	// CursorOccurredAt / CursorEventID 游标位置（不含）
	CursorOccurredAt int64
	CursorEventID    string
	HasCursor        bool
}

// EventModel 只读查询 message_events。
type EventModel interface {
	// FindByMessageID 消息时间线
	FindByMessageID(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error)
	// FindByReceiver 投递/连接侧时间线
	FindByReceiver(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error)
	// FindBySender 连接事件（senderId=被观察用户）
	FindBySender(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error)
	// HasAnyByMessageID 是否存在该消息的任意事件
	HasAnyByMessageID(ctx context.Context, messageID string) (bool, error)
	// CountGaps 统计窗口内 observation_gap 丢弃计数
	CountGaps(ctx context.Context, startTime, endTime int64) (uint64, bool, error)
	// FindOneByID 按 eventId 查单条
	FindOneByID(ctx context.Context, eventID string) (*observation.MessageEvent, error)
}

type defaultEventModel struct {
	coll *mongo.Collection
}

// NewEventModel 打开 message_events 只读模型。索引由写侧 EnsureIndexes 负责。
func NewEventModel(url, db string) (EventModel, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(url))
	if err != nil {
		return nil, fmt.Errorf("operationsmodels: connect mongo: %w", err)
	}
	return &defaultEventModel{coll: client.Database(db).Collection(observation.CollectionMessageEvents)}, nil
}

// MustEventModel 启动期构建，失败即 panic。
func MustEventModel(url, db string) EventModel {
	m, err := NewEventModel(url, db)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *defaultEventModel) FindByMessageID(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error) {
	filter := bson.M{"messageId": f.MessageID}
	applyCommon(filter, f)
	return m.query(ctx, filter, f)
}

func (m *defaultEventModel) FindByReceiver(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error) {
	filter := bson.M{}
	if f.MessageID != "" {
		filter["messageId"] = f.MessageID
	}
	if f.ReceiverID != "" {
		filter["receiverId"] = f.ReceiverID
	}
	applyCommon(filter, f)
	return m.query(ctx, filter, f)
}

func (m *defaultEventModel) FindBySender(ctx context.Context, f EventQueryFilter) ([]*observation.MessageEvent, error) {
	// 连接事件约定：senderId = 被观察 userId
	filter := bson.M{"senderId": f.ReceiverID}
	if f.ReceiverID == "" {
		return nil, fmt.Errorf("operationsmodels: userId is required")
	}
	if len(f.EventTypes) == 0 {
		filter["eventType"] = bson.M{"$in": []string{observation.EventOnline, observation.EventOffline}}
	}
	applyCommon(filter, f)
	return m.query(ctx, filter, f)
}

func (m *defaultEventModel) HasAnyByMessageID(ctx context.Context, messageID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var raw bson.M
	err := m.coll.FindOne(ctx, bson.M{"messageId": messageID}, options.FindOne().SetProjection(bson.M{"eventId": 1})).Decode(&raw)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *defaultEventModel) CountGaps(ctx context.Context, startTime, endTime int64) (uint64, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	filter := bson.M{"eventType": observation.EventObservationGap}
	if startTime > 0 || endTime > 0 {
		rng := bson.M{}
		if startTime > 0 {
			rng["$gte"] = startTime
		}
		if endTime > 0 {
			rng["$lte"] = endTime
		}
		filter["occurredAt"] = rng
	}

	cur, err := m.coll.Find(ctx, filter)
	if err != nil {
		return 0, false, err
	}
	defer cur.Close(ctx)

	var dropped uint64
	found := false
	for cur.Next(ctx) {
		var ev observation.MessageEvent
		if err := cur.Decode(&ev); err != nil {
			return 0, false, err
		}
		found = true
		if ev.Metadata != nil {
			var n uint64
			if _, err := fmt.Sscanf(ev.Metadata["droppedCount"], "%d", &n); err == nil {
				dropped += n
			}
		}
	}
	return dropped, found, cur.Err()
}

func (m *defaultEventModel) FindOneByID(ctx context.Context, eventID string) (*observation.MessageEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var ev observation.MessageEvent
	err := m.coll.FindOne(ctx, bson.M{"eventId": eventID}).Decode(&ev)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ev, nil
}

func applyCommon(filter bson.M, f EventQueryFilter) {
	if len(f.EventTypes) == 1 {
		filter["eventType"] = f.EventTypes[0]
	} else if len(f.EventTypes) > 1 {
		filter["eventType"] = bson.M{"$in": f.EventTypes}
	}

	rng := bson.M{}
	if f.StartTime > 0 {
		rng["$gte"] = f.StartTime
	}
	if f.EndTime > 0 {
		rng["$lte"] = f.EndTime
	}
	if len(rng) > 0 {
		filter["occurredAt"] = rng
	}

	if f.HasCursor {
		// (occurredAt, eventId) 严格大于游标
		filter["$or"] = []bson.M{
			{"occurredAt": bson.M{"$gt": f.CursorOccurredAt}},
			{"occurredAt": f.CursorOccurredAt, "eventId": bson.M{"$gt": f.CursorEventID}},
		}
	}
}

func (m *defaultEventModel) query(ctx context.Context, filter bson.M, f EventQueryFilter) ([]*observation.MessageEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "occurredAt", Value: 1}, {Key: "eventId", Value: 1}}).
		SetLimit(limit)

	cur, err := m.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*observation.MessageEvent
	for cur.Next(ctx) {
		var ev observation.MessageEvent
		if err := cur.Decode(&ev); err != nil {
			return nil, err
		}
		out = append(out, &ev)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

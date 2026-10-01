package msgtransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/ws"
	"github.com/IM_System/apps/task/mq/internal/svc"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/bitmap"
	"github.com/IM_System/pkg/observation"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MsgChatTransfer struct {
	*baseMsgTransfer
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{NewBaseMsgTransfer(svc)}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) (consumeErr error) { // 消费者，有更新消息时会调用这个方法
	if !m.svcCtx.Config.LoadTest.PersistenceOnly {
		fmt.Printf("consume msg, key: %s, value: %s\n", key, value)
	}

	var data mq.MsgChatTransfer

	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}
	throttleStarted := time.Now()
	if m.svcCtx.ChatLimiter != nil {
		if err := m.svcCtx.ChatLimiter.Wait(ctx); err != nil {
			return err
		}
	}
	m.svcCtx.Metrics.ObserveThrottle(time.Since(throttleStarted))
	started := time.Now()
	if m.svcCtx.Config.LoadTest.PersistenceOnly {
		defer func() {
			finished := time.Now()
			metric := struct {
				Type            string  `json:"type"`
				ConversationID  string  `json:"conversationId"`
				MessageID       string  `json:"messageId"`
				ProcessingMs    float64 `json:"processingMs"`
				ThrottleWaitMs  float64 `json:"throttleWaitMs"`
				ToPersistedMs   float64 `json:"toPersistedMs"`
				CompletedUnixMs int64   `json:"completedUnixMs"`
				Success         bool    `json:"success"`
			}{"loadtest_message", data.ConversationId, data.MessageId,
				float64(finished.Sub(started).Microseconds()) / 1000,
				float64(started.Sub(throttleStarted).Microseconds()) / 1000,
				float64(finished.Sub(time.Unix(0, data.SendTime)).Microseconds()) / 1000,
				finished.UnixMilli(), consumeErr == nil}
			encoded, _ := json.Marshal(metric)
			fmt.Println(string(encoded))
		}()
	}

	// 新消息沿用 im-ws 生成的稳定 messageId；旧消息（无 messageId）保持消费端生成旧行为
	msgId, legacy := mq.ResolveMessageId(&data)
	msgIdHex := msgId.Hex()
	source := observation.SourceTaskMq
	if legacy {
		source = observation.SourceCompatLegacy
	}

	builder := observation.MessageEventBuilder{
		MessageID:       msgIdHex,
		ClientMessageID: data.ClientMessageId,
		CorrelationID:   mq.NormalizeCorrelationId(&data, msgIdHex),
		ConversationID:  data.ConversationId,
		SenderID:        data.SendId,
		ReceiverID:      data.RecvId,
		Source:          source,
	}

	observation.SafeRecord(ctx, m.svcCtx.ObservationSink, builder.KafkaConsumed())

	// 记录数据
	inserted, err := m.addChatLog(ctx, msgId, data)
	if err != nil {
		observation.SafeRecord(ctx, m.svcCtx.ObservationSink, builder.PersistFailed(observation.ErrCodePersistFailed))
		return err
	}
	if !inserted {
		m.svcCtx.Metrics.IncDuplicate()
	}

	observation.SafeRecord(ctx, m.svcCtx.ObservationSink, builder.Persisted())

	return m.Transfer(ctx, &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		RecvIds:        data.RecvIds,
		MsgId:          msgIdHex,
		SendTime:       data.SendTime,
		MType:          data.MType,
		Content:        data.Content,
	})
}

func (m *MsgChatTransfer) addChatLog(ctx context.Context, msgId bson.ObjectID, data mq.MsgChatTransfer) (bool, error) {
	chatLog := &immodels.ChatLog{
		ID:             msgId,
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       data.SendTime,
	}

	readRecords := bitmap.NewBitmap(0)
	readRecords.Set(chatLog.SendId) // 发送者默认已读
	chatLog.ReadRecords = readRecords.Export()

	inserted, err := m.svcCtx.ChatLogModel.InsertIfAbsent(ctx, chatLog)
	if err != nil {
		return false, err
	}
	if !inserted {
		return false, nil
	}
	return true, m.svcCtx.ConversationModel.UpdateMsg(ctx, chatLog)
}

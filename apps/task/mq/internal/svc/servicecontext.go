package svc

import (
	"context"
	"net/http"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/apps/task/mq/internal/dlq"
	"github.com/IM_System/apps/task/mq/internal/telemetry"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/observation"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

var retryPolicy = `{
  "methodConfig": [{
    "name": [{"service": "social.Social"}],
    "waitForReady": true,
    "retryPolicy": {
      "MaxAttempts": 5,
      "InitialBackoff": "0.001s",
      "MaxBackoff": "0.002s",
      "BackoffMultiplier": 1.0,
      "RetryableStatusCodes": ["UNKNOWN","DEADLINE_EXCEEDED"]
    }
  }]
}`

type ServiceContext struct {
	config.Config

	WsClient websocket.Client

	*redis.Redis

	immodels.ChatLogModel

	immodels.ConversationModel

	immodels.ConversationsModel

	socialclient.Social

	// ObservationSink 旁路观测；默认 Noop，失败不影响业务
	ObservationSink observation.ObservationSink
	ChatLimiter     *rate.Limiter
	Metrics         *telemetry.Metrics
	DeadLetter      dlq.Publisher
}

func NewServiceContext(c config.Config) *ServiceContext {
	svc := &ServiceContext{
		Config: c,

		ChatLogModel: immodels.MustChatLogModel(c.Mongo.Url, c.Mongo.Db),

		ConversationModel: immodels.MustConversationModel(c.Mongo.Url, c.Mongo.Db),

		ConversationsModel: immodels.MustConversationsModel(c.Mongo.Url, c.Mongo.Db),

		ObservationSink: newObservationSink(c),
		Metrics:         telemetry.NewMetrics(),
	}
	if c.MessageProcessingRateLimit.Enabled {
		perInstance := c.PerInstanceMessagesPerSecond()
		if perInstance <= 0 {
			panic("MessageProcessingRateLimit must resolve to a positive per-instance rate")
		}
		svc.ChatLimiter = rate.NewLimiter(rate.Limit(perInstance), c.MessageProcessingRateLimit.Burst)
	} else if c.LoadTest.PersistenceOnly && c.LoadTest.MaxMessagesPerSecond > 0 {
		// Backwards-compatible path for the untracked isolated load-test harness.
		svc.ChatLimiter = rate.NewLimiter(rate.Limit(c.LoadTest.MaxMessagesPerSecond), c.LoadTest.Burst)
	}
	if c.MessageRetry.Enabled {
		publisher, err := dlq.NewKafkaPublisher(c.MsgChatTransfer, c.MessageRetry.DeadLetterTopic)
		if err != nil {
			panic(err)
		}
		svc.DeadLetter = publisher
	}
	if c.LoadTest.PersistenceOnly {
		return svc
	}
	svc.Redis = redis.MustNewRedis(c.Redisx)
	svc.Social = socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
		zrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy))))

	token, err := svc.GetSystemToken()
	if err != nil {
		panic(err)
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	svc.WsClient = websocket.NewClient(c.Ws.Host, websocket.WithClientHeader(header))

	return svc
}
func (svc *ServiceContext) GetSystemToken() (string, error) {
	return svc.Redis.Get(constants.REDIS_SYSTEM_ROOT_TOKEN)
}

func newObservationSink(c config.Config) observation.ObservationSink {
	cfg := observation.DeliveryObservation{
		Enabled:       c.DeliveryObservation.Enabled,
		PersistEvents: c.DeliveryObservation.PersistEvents,
		AckMode:       c.DeliveryObservation.AckMode,
		InstanceId:    c.DeliveryObservation.InstanceId,
		BufferSize:    c.DeliveryObservation.BufferSize,
	}
	if !cfg.Enabled {
		return observation.Nop()
	}
	if !cfg.PersistEvents {
		cfg.PersistEvents = true
	}

	sink, _, err := observation.NewSinkWithSource(
		context.Background(),
		cfg,
		c.Mongo.Url,
		c.Mongo.Db,
		observation.SourceTaskMq,
	)
	if err != nil {
		return observation.Nop()
	}
	return sink
}

package config

import (
	"fmt"
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf

	ListenOn string

	MsgChatTransfer kq.KqConf
	MsgReadTransfer kq.KqConf

	Redisx redis.RedisConf

	Mongo struct {
		Url string
		Db  string
	}

	SocialRpc zrpc.RpcClientConf

	Ws struct {
		Host string
	}

	// MessageProcessingRateLimit protects downstream storage. It limits business
	// message handling, not Kafka fetching. The global budget is divided evenly
	// across the expected number of task-mq instances.
	MessageProcessingRateLimit MessageProcessingRateLimitConfig `json:",optional"`

	// LoadTest is retained only for the isolated test harness. Production
	// deployments must use MessageProcessingRateLimit above.
	LoadTest struct {
		PersistenceOnly      bool `json:",default=false"`
		MaxMessagesPerSecond int  `json:",default=0"`
		Burst                int  `json:",default=0"`
	} `json:",optional"`

	KafkaMonitoring struct {
		Enabled        bool   `json:",default=false"`
		ListenOn       string `json:",default=0.0.0.0:10092"`
		LagIntervalSec int    `json:",default=10"`
	} `json:",optional"`

	MessageRetry MessageRetryConfig `json:",optional"`

	ConsumerScaling struct {
		ExpectedPartitions int `json:",default=1"`
	} `json:",optional"`

	MsgReadHandler struct {
		GroupMsgReadHandler          int
		GroupMsgReadRecordDelayTime  int64
		GroupMsgReadRecordDelayCount int
	}

	// DeliveryObservation 观测开关，默认关闭。第一版修改需重启，不做热更新。
	DeliveryObservation struct {
		Enabled       bool
		PersistEvents bool
		AckMode       string
		InstanceId    string
		BufferSize    int
	}
}

type MessageProcessingRateLimitConfig struct {
	Enabled                 bool `json:",default=false"`
	GlobalMessagesPerSecond int  `json:",default=0"`
	ExpectedInstances       int  `json:",default=1"`
	Burst                   int  `json:",default=1"`
}

type MessageRetryConfig struct {
	Enabled          bool   `json:",default=false"`
	MaxAttempts      int    `json:",default=3"`
	InitialBackoffMs int    `json:",default=100"`
	MaxBackoffMs     int    `json:",default=2000"`
	DeadLetterTopic  string `json:",optional"`
}

func (c Config) Validate() error {
	rateLimit := c.MessageProcessingRateLimit
	if rateLimit.Enabled {
		if rateLimit.GlobalMessagesPerSecond <= 0 {
			return fmt.Errorf("MessageProcessingRateLimit.GlobalMessagesPerSecond must be positive")
		}
		if rateLimit.ExpectedInstances <= 0 {
			return fmt.Errorf("MessageProcessingRateLimit.ExpectedInstances must be positive")
		}
		if rateLimit.Burst <= 0 {
			return fmt.Errorf("MessageProcessingRateLimit.Burst must be positive")
		}
	}
	if c.MessageRetry.Enabled {
		if c.MessageRetry.MaxAttempts <= 0 {
			return fmt.Errorf("MessageRetry.MaxAttempts must be positive")
		}
		if c.MessageRetry.InitialBackoffMs <= 0 || c.MessageRetry.MaxBackoffMs <= 0 ||
			c.MessageRetry.InitialBackoffMs > c.MessageRetry.MaxBackoffMs {
			return fmt.Errorf("MessageRetry backoff values are invalid")
		}
		if c.MessageRetry.DeadLetterTopic == "" {
			return fmt.Errorf("MessageRetry.DeadLetterTopic is required")
		}
	}
	if c.KafkaMonitoring.Enabled {
		if c.KafkaMonitoring.ListenOn == "" || c.KafkaMonitoring.LagIntervalSec <= 0 {
			return fmt.Errorf("KafkaMonitoring listen address and interval are required")
		}
	}
	if c.ConsumerScaling.ExpectedPartitions < 0 {
		return fmt.Errorf("ConsumerScaling.ExpectedPartitions must not be negative")
	}
	return nil
}

func (c Config) PerInstanceMessagesPerSecond() float64 {
	if !c.MessageProcessingRateLimit.Enabled {
		return 0
	}
	return float64(c.MessageProcessingRateLimit.GlobalMessagesPerSecond) /
		float64(c.MessageProcessingRateLimit.ExpectedInstances)
}

func (c Config) RetryInitialBackoff() time.Duration {
	return time.Duration(c.MessageRetry.InitialBackoffMs) * time.Millisecond
}

func (c Config) RetryMaxBackoff() time.Duration {
	return time.Duration(c.MessageRetry.MaxBackoffMs) * time.Millisecond
}

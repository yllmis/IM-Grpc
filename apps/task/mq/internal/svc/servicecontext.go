package svc

import (
	"net/http"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/task/mq/internal/config"
	"github.com/IM_System/pkg/constants"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	config.Config

	WsClient websocket.Client

	*redis.Redis

	immodels.ChatLogModel

	immodels.ConversationModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	svc := &ServiceContext{
		Config: c,

		Redis: redis.MustNewRedis(c.Redisx),

		ChatLogModel: immodels.MustChatLogModel(c.Mongo.Url, c.Mongo.Db),

		ConversationModel: immodels.MustConversationModel(c.Mongo.Url, c.Mongo.Db),
	}

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

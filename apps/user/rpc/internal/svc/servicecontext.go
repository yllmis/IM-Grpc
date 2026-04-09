package svc

import (
	"time"

	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/config"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	models.UsersModel
	Redisx *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.Mysql.DataSource)

	return &ServiceContext{
		Config: c,

		Redisx: redis.MustNewRedis(c.Redisx),

		UsersModel: models.NewUsersModel(sqlConn, c.Cache),
	}
}

func (svc *ServiceContext) SetRootToken() error {
	// 生成一个jwt，存储在 Redis 中，并设置过期时间长
	systemToken, err := ctxdata.GetTokenKey(svc.Config.JwtAuth.AccessSecret, time.Now().Unix(), 999999999, constants.SYSTEM_ROOT_ID)
	if err != nil {
		return err
	}

	return svc.Redisx.Set(constants.REDIS_SYSTEM_ROOT_TOKEN, systemToken)

}

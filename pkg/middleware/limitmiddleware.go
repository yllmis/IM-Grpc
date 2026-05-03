package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type LimitMiddleware struct {
	redisConf           redis.RedisConf
	*limit.TokenLimiter // 令牌桶算法的限流器
}

func NewLimitMiddleware(cfg redis.RedisConf) *LimitMiddleware {
	return &LimitMiddleware{redisConf: cfg}
}

func (m *LimitMiddleware) TokenLimitHandler(rate, burst int) rest.Middleware {
	m.TokenLimiter = limit.NewTokenLimiter(rate, burst, redis.MustNewRedis(m.redisConf), "REDIS_TOKEN_LIMIT_KEY")

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if m.TokenLimiter.AllowCtx(r.Context()) {
				next(w, r)
				return
			}

		}
	}
}

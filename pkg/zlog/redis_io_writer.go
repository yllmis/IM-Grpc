package zlog

import (
	"io"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type redisIoWriter struct {
	redisKey string
	redis    *redis.Redis
}

func NewRedisIoWriter(redisKey string, rediscfg redis.RedisConf) io.Writer {
	return &redisIoWriter{
		redisKey: redisKey,
		redis:    redis.MustNewRedis(rediscfg),
	}
}

func (r *redisIoWriter) Write(p []byte) (n int, err error) {
	// 写日志
	go r.redis.Rpush(r.redisKey, string(p))

	return 0, nil
}

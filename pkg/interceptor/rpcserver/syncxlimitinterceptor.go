package rpcserver

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SyncxLimitInterceptor(maxCount int) grpc.UnaryServerInterceptor {
	l := syncx.NewLimit(maxCount)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if l.TryBorrow() {
			defer func() {
				if err := l.Return(); err != nil {
					// 这里可以记录日志，说明返回令牌失败了
					logx.Error(err)
				}
			}()
			return handler(ctx, req)
		} else {
			logx.Errorf("concurrent connection over %d, rejected with code %d",
				maxCount, http.StatusServiceUnavailable)
			return nil, status.Error(codes.Unavailable, "oncurrent connections over limit")
		}
	}
}

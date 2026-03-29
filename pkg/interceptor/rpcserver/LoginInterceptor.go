package rpcserver

import (
	"context"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	zerr "github.com/zeromicro/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoginInterceptorfunc(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	resp, err = handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	// 如果是自定义的错误，要转换为grpc接收的错误格式
	// 这里的转换逻辑根据实际情况进行调整
	// 例如，可以使用grpc/status包来创建一个新的错误，包含原始错误的信息
	logx.WithContext(ctx).Errorf("【RPC SRV ERR】 %v", err) // 记录错误日志，方便排查问题

	causeErr := errors.Cause(err)              // 获取原始错误
	if e, ok := causeErr.(*zerr.CodeMsg); ok { // 判断是否是自定义的错误类型
		err = status.Error(codes.Code(e.Code), e.Msg) // 转换为grpc错误格式
	}
	return resp, err
}

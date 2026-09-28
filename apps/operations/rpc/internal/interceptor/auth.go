package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MetadataServiceToken 服务身份 token 的 metadata key。
const MetadataServiceToken = "x-im-service-token"

// Auth 服务身份校验（docs/operations-query.md §3.5）。
// 权限不足一律 PERMISSION_DENIED，不泄露资源是否存在。
type Auth struct {
	Enable bool
	Tokens map[string]struct{}
}

// NewAuth 构建校验器。tokens 含主 token 与轮换重叠窗口的旧 token。
func NewAuth(enable bool, tokens ...string) *Auth {
	set := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t != "" {
			set[t] = struct{}{}
		}
	}
	return &Auth{Enable: enable, Tokens: set}
}

// UnaryInterceptor 统一在 interceptor 校验，不把身份交给业务 logic。
func (a *Auth) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if a.Enable {
		if err := a.verify(ctx); err != nil {
			return nil, err
		}
	}
	return handler(ctx, req)
}

func (a *Auth) verify(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.PermissionDenied, "missing service credentials")
	}

	vals := md.Get(MetadataServiceToken)
	if len(vals) == 0 {
		return status.Error(codes.PermissionDenied, "missing service credentials")
	}

	// token、校验详情不得写入日志
	for _, v := range vals {
		if _, ok := a.Tokens[strings.TrimSpace(v)]; ok {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "invalid service credentials")
}

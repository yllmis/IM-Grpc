package interceptor

import (
	"context"
	"testing"

	"github.com/IM_System/pkg/serviceauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestIdempotenceKeepsBusinessCredentialAndTraceMetadata(t *testing.T) {
	for _, taskID := range []string{"", "request-test-only"} {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("trace-id", "trace-test-only"))
		if taskID != "" {
			ctx = context.WithValue(ctx, Tkey, taskID)
		}
		business := serviceauth.BusinessClientInterceptor("backend-test-only")
		idempotent := NewIdempotentClient(DefaultIdempotent)
		const method = "/social.social/GroupCreate"
		err := business(ctx, method, nil, nil, nil, func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return idempotent(ctx, method, req, reply, cc, func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				md, _ := metadata.FromOutgoingContext(ctx)
				if md.Get("trace-id")[0] != "trace-test-only" {
					t.Fatal("trace metadata dropped")
				}
				if taskID == "" && len(md.Get(Dkey)) != 0 {
					t.Fatal("empty global idempotence key emitted")
				}
				if taskID != "" && (len(md.Get(Dkey)) != 1 || md.Get(Dkey)[0] != taskID+"."+method) {
					t.Fatal("invalid dispatch key")
				}
				guard := serviceauth.Guard(serviceauth.Config{Enable: true, Token: "backend-test-only"}, "social.social")
				_, err := guard(metadata.NewIncomingContext(context.Background(), md), nil, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) { return nil, nil })
				return err
			}, opts...)
		})
		if err != nil {
			t.Fatalf("credential lost across client chain: %v", err)
		}
		original, _ := metadata.FromOutgoingContext(ctx)
		if len(original.Get(serviceauth.MetadataServiceToken)) != 0 || len(original.Get(Dkey)) != 0 {
			t.Fatal("original context mutated")
		}
	}
}

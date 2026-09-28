package interceptor

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func handlerOK(ctx context.Context, req any) (any, error) {
	return "ok", nil
}

func TestAuth_Disabled(t *testing.T) {
	a := NewAuth(false)
	resp, err := a.UnaryInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handlerOK)
	if err != nil || resp != "ok" {
		t.Fatalf("disabled auth must pass: %v %v", resp, err)
	}
}

func TestAuth_MissingToken(t *testing.T) {
	a := NewAuth(true, "secret")
	_, err := a.UnaryInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handlerOK)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("missing token must PERMISSION_DENIED, got %v", err)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	a := NewAuth(true, "secret")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataServiceToken, "wrong"))
	_, err := a.UnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handlerOK)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("invalid token must PERMISSION_DENIED, got %v", err)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	a := NewAuth(true, "secret")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataServiceToken, "secret"))
	resp, err := a.UnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handlerOK)
	if err != nil || resp != "ok" {
		t.Fatalf("valid token must pass: %v %v", resp, err)
	}
}

func TestAuth_RotationOverlap(t *testing.T) {
	a := NewAuth(true, "new-token", "old-token")
	for _, tok := range []string{"new-token", "old-token"} {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataServiceToken, tok))
		if _, err := a.UnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, handlerOK); err != nil {
			t.Fatalf("token %s must pass during rotation: %v", tok, err)
		}
	}
}

package serviceauth

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestGuardFailsClosedAndDoesNotChangeLegacyMethods(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		guard := Guard(Config{Enable: enabled, Token: "domain-secret"}, "im.MessageQuery")
		called := false
		handler := func(context.Context, any) (any, error) { called = true; return "ok", nil }
		_, err := guard(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/im.MessageQuery/GetMessageRecord"}, handler)
		if status.Code(err) != codes.PermissionDenied || called {
			t.Fatalf("enabled=%v %v", enabled, err)
		}
		resp, err := guard(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/im.Im/GetChatLog"}, handler)
		if err != nil || resp != "ok" {
			t.Fatalf("legacy behavior changed: %v", err)
		}
	}
}
func TestGuardRotationAndTokenReplacement(t *testing.T) {
	guard := Guard(Config{Enable: true, Token: "new", ExtraTokens: []string{"old"}}, "user.UserQuery")
	for _, token := range []string{"old", "new"} {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataServiceToken, token))
		_, err := guard(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/user.UserQuery/FindUserReference"}, func(context.Context, any) (any, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(MetadataServiceToken, "caller", "trace-id", "trace"))
	forwarded := Outgoing(ctx, "domain")
	original, _ := metadata.FromOutgoingContext(ctx)
	md, _ := metadata.FromOutgoingContext(forwarded)
	if original.Get(MetadataServiceToken)[0] != "caller" || len(md.Get(MetadataServiceToken)) != 1 || md.Get(MetadataServiceToken)[0] != "domain" || md.Get("trace-id")[0] != "trace" {
		t.Fatal("credential replacement mutated metadata")
	}
}
func TestEnabledAuthRequiresPrimaryToken(t *testing.T) {
	if err := (Config{Enable: true}).Validate(); err == nil {
		t.Fatal("missing token accepted")
	}
	if err := (Config{}).Validate(); err != nil {
		t.Fatal(err)
	}
}

package serviceauth

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestBusinessEnvironmentFailsClosed(t *testing.T) {
	for _, value := range []string{"", " "} {
		t.Setenv(BusinessTokenEnvironment, value)
		if _, err := BusinessConfig(); err == nil || strings.Contains(err.Error(), "actual-secret") {
			t.Fatal("empty credential accepted")
		}
	}
	t.Setenv(BusinessTokenEnvironment, " actual-secret ")
	c, err := BusinessConfig()
	if err != nil || !c.Enable || c.Token != "actual-secret" {
		t.Fatalf("invalid configuration: %v", err)
	}
}

func TestBusinessAndQueryCredentialSeparation(t *testing.T) {
	business := Config{Enable: true, Token: "backend"}
	for _, query := range []Config{
		{Enable: true, Token: "backend"},
		{Enable: true, Token: "read-only", ExtraTokens: []string{"backend"}},
	} {
		if err := RequireSeparateCredentials(business, query); err == nil {
			t.Fatal("read-only credential can become privileged")
		}
	}
	if err := RequireSeparateCredentials(business, Config{Enable: true, Token: "read-only"}); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessAndReadOnlyGuardsRejectCrossUse(t *testing.T) {
	for _, tc := range []struct{ service, method string }{
		{"im.Im", "GetChatLog"}, {"user.User", "GetUserInfo"}, {"social.social", "Groupusers"},
	} {
		guard := Guard(Config{Enable: true, Token: "backend"}, tc.service)
		for _, credential := range []string{"", "read-only", "ordinary-user-jwt", "backend"} {
			called := false
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(MetadataServiceToken, credential))
			_, err := guard(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/" + tc.service + "/" + tc.method}, func(context.Context, any) (any, error) {
				called = true
				return nil, nil
			})
			if credential == "backend" {
				if err != nil || !called {
					t.Fatal("backend rejected")
				}
			} else if status.Code(err) != codes.PermissionDenied || called {
				t.Fatal("unauthorized request reached business handler")
			}
		}
	}
}

func TestBusinessClientDoesNotLeakCredentials(t *testing.T) {
	i := BusinessClientInterceptor("backend")
	for _, method := range []string{"/im.Im/GetChatLog", "/user.User/Login", "/social.social/Groupusers", "/im.MessageQuery/GetMessageRecord", "/other.Service/Call"} {
		ctx := Outgoing(context.Background(), "query")
		err := i(ctx, method, nil, nil, nil, func(ctx context.Context, m string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			want := "query"
			if strings.HasPrefix(method, "/im.Im/") || strings.HasPrefix(method, "/user.User/") || strings.HasPrefix(method, "/social.social/") {
				want = "backend"
			}
			if got := md.Get(MetadataServiceToken); len(got) != 1 || got[0] != want {
				t.Fatal("credential routing is incorrect")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		original, _ := metadata.FromOutgoingContext(ctx)
		if original.Get(MetadataServiceToken)[0] != "query" {
			t.Fatal("mutated caller context")
		}
	}
}

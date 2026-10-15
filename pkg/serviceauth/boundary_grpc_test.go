package serviceauth

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/user/rpc/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type boundaryIm struct{ im.UnimplementedImServer }

func (*boundaryIm) GetChatLog(context.Context, *im.GetChatLogReq) (*im.GetChatLogResp, error) {
	return &im.GetChatLogResp{}, nil
}

type boundaryUser struct{ user.UnimplementedUserServer }

func (*boundaryUser) GetUserInfo(context.Context, *user.GetUserInfoReq) (*user.GetUserInfoResp, error) {
	return &user.GetUserInfoResp{}, nil
}

type boundarySocial struct {
	social.UnimplementedSocialServer
}

func (*boundarySocial) Groupusers(context.Context, *social.GroupusersReq) (*social.GroupusersResp, error) {
	return &social.GroupusersResp{}, nil
}

type boundaryQuery struct {
	im.UnimplementedMessageQueryServer
}

func (*boundaryQuery) GetMessageRecord(context.Context, *im.MessageRecordRequest) (*im.MessageRecordResponse, error) {
	return &im.MessageRecordResponse{}, nil
}

func TestGRPCBusinessAndQueryBoundary(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	business := Config{Enable: true, Token: "backend-test-only"}
	query := Config{Enable: true, Token: "query-test-only"}
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		Guard(business, "im.Im"), Guard(business, "user.User"), Guard(business, "social.social"), Guard(query, "im.MessageQuery"),
	))
	im.RegisterImServer(server, &boundaryIm{})
	user.RegisterUserServer(server, &boundaryUser{})
	social.RegisterSocialServer(server, &boundarySocial{})
	im.RegisterMessageQueryServer(server, &boundaryQuery{})
	go server.Serve(lis)
	t.Cleanup(func() { server.Stop(); lis.Close() })
	conn, err := grpc.NewClient("passthrough:///permissions-test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := []struct {
		name  string
		query bool
		call  func(context.Context) error
	}{
		{"business messages", false, func(ctx context.Context) error {
			_, err := im.NewImClient(conn).GetChatLog(ctx, &im.GetChatLogReq{})
			return err
		}},
		{"business users", false, func(ctx context.Context) error {
			_, err := user.NewUserClient(conn).GetUserInfo(ctx, &user.GetUserInfoReq{})
			return err
		}},
		{"business social", false, func(ctx context.Context) error {
			_, err := social.NewSocialClient(conn).Groupusers(ctx, &social.GroupusersReq{})
			return err
		}},
		{"read-only messages", true, func(ctx context.Context) error {
			_, err := im.NewMessageQueryClient(conn).GetMessageRecord(ctx, &im.MessageRecordRequest{})
			return err
		}},
	}
	for _, c := range calls {
		for _, token := range []string{"", "invalid", "user-jwt-test-only", business.Token, query.Token} {
			requestCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(MetadataServiceToken, token, "authorization", "Bearer user-jwt-test-only"))
			err := c.call(requestCtx)
			allowed := (!c.query && token == business.Token) || (c.query && token == query.Token)
			if allowed && err != nil {
				t.Fatalf("%s normal access failed: %v", c.name, err)
			}
			if !allowed && status.Code(err) != codes.PermissionDenied {
				t.Fatalf("%s crossed credential boundary: %v", c.name, err)
			}
		}
	}
}

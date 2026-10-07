package server

import (
	"context"
	"errors"
	"github.com/IM_System/apps/user/models"
	"github.com/IM_System/apps/user/rpc/internal/svc"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/serviceauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

type fakeReferenceModel struct {
	models.UsersModel
	rows   []models.UserReferenceRow
	err    error
	filter models.UserReferenceFilter
	calls  int
}

func (f *fakeReferenceModel) SearchReferences(ctx context.Context, q models.UserReferenceFilter) ([]models.UserReferenceRow, error) {
	f.calls++
	f.filter = q
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing query budget")
	}
	return f.rows, f.err
}
func TestUserReferenceContract(t *testing.T) {
	model := &fakeReferenceModel{rows: []models.UserReferenceRow{{ID: "u1", DisplayName: "tester"}, {ID: "u2", DisplayName: "tester"}}}
	s := NewUserQueryServer(&svc.ServiceContext{UsersModel: model})
	resp, err := s.FindUserReference(context.Background(), &user.UserReferenceRequest{Nickname: "tester", Limit: 1})
	if err != nil || len(resp.Users) != 1 || !resp.Truncated || model.filter.Limit != 1 {
		t.Fatalf("%+v %v", resp, err)
	}
	fields := resp.Users[0].ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		if name == "password" || name == "phone" || name == "avatar" || name == "token" {
			t.Fatal("sensitive field", name)
		}
	}
	for _, req := range []*user.UserReferenceRequest{nil, {}, {UserId: "u1", Phone: "123"}, {UserId: "u1", Limit: 201}, {Nickname: "  "}, {UserId: "u1", Limit: -1}} {
		if _, err := s.FindUserReference(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("%+v %v", req, err)
		}
	}
	model.rows = nil
	resp, err = s.FindUserReference(context.Background(), &user.UserReferenceRequest{Phone: "123"})
	if err != nil || len(resp.Users) != 0 || model.filter.Limit != 50 {
		t.Fatalf("%+v %v", resp, err)
	}
	model.err = context.DeadlineExceeded
	resp, err = s.FindUserReference(context.Background(), &user.UserReferenceRequest{Phone: "123"})
	if resp != nil || status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("phone DB failure swallowed: %+v %v", resp, err)
	}
}
func TestUserQueryAuthenticatedGRPC(t *testing.T) {
	model := &fakeReferenceModel{}
	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer(grpc.UnaryInterceptor(serviceauth.Guard(serviceauth.Config{Enable: true, Token: "domain-secret"}, "user.UserQuery")))
	user.RegisterUserQueryServer(gs, NewUserQueryServer(&svc.ServiceContext{UsersModel: model}))
	go gs.Serve(lis)
	t.Cleanup(func() { gs.Stop(); lis.Close() })
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := user.NewUserQueryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &user.UserReferenceRequest{UserId: "u1"}
	if _, err := client.FindUserReference(ctx, req); status.Code(err) != codes.PermissionDenied || model.calls != 0 {
		t.Fatalf("auth: %v", err)
	}
	resp, err := client.FindUserReference(serviceauth.Outgoing(ctx, "domain-secret"), req)
	if err != nil || len(resp.Users) != 0 || model.calls != 1 {
		t.Fatalf("%+v %v", resp, err)
	}
}

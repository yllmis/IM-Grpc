package server

import (
	"context"
	"errors"
	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/serviceauth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

type fakeRecordModel struct {
	immodels.ChatLogModel
	row   *immodels.ChatLog
	err   error
	calls int
}

func (f *fakeRecordModel) FindOne(ctx context.Context, _ string) (*immodels.ChatLog, error) {
	f.calls++
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing query budget")
	}
	return f.row, f.err
}

type fakeSearchModel struct {
	rows   []immodels.MessageReference
	err    error
	filter immodels.MessageSearchFilter
}

func (f *fakeSearchModel) Search(_ context.Context, q immodels.MessageSearchFilter) ([]immodels.MessageReference, error) {
	f.filter = q
	return f.rows, f.err
}
func TestMessageRecordContract(t *testing.T) {
	oid := bson.NewObjectID()
	for _, tc := range []struct {
		name       string
		storageErr error
		code       codes.Code
		found      bool
	}{
		{"found", nil, codes.OK, true}, {"missing", immodels.ErrNotFound, codes.OK, false},
		{"timeout", context.DeadlineExceeded, codes.DeadlineExceeded, false},
		{"storage", errors.New("internal database secret"), codes.Internal, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := &fakeRecordModel{row: &immodels.ChatLog{ID: oid, SendId: "u1", RecvId: "u2", SendTime: 42, ChatType: constants.SingleChatType, MsgContent: "must-not-leak", ReadRecords: []byte{1}}, err: tc.storageErr}
			resp, err := NewMessageQueryServer(&svc.ServiceContext{ChatLogModel: model}).GetMessageRecord(context.Background(), &im.MessageRecordRequest{MessageId: oid.Hex()})
			if status.Code(err) != tc.code {
				t.Fatalf("code=%v err=%v", tc.code, err)
			}
			if err != nil {
				if resp != nil {
					t.Fatal("error must not become empty success")
				}
				return
			}
			if resp.Found != tc.found || resp.MessageId != oid.Hex() || resp.ObservedAt <= 0 {
				t.Fatalf("%+v", resp)
			}
			fields := resp.ProtoReflect().Descriptor().Fields()
			for _, name := range []string{"msgContent", "readRecords", "eventsAvailable", "eventsState"} {
				for i := 0; i < fields.Len(); i++ {
					if string(fields.Get(i).Name()) == name {
						t.Fatal("domain response contains forbidden field", name)
					}
				}
			}
		})
	}
	model := &fakeRecordModel{}
	_, err := NewMessageQueryServer(&svc.ServiceContext{ChatLogModel: model}).GetMessageRecord(context.Background(), &im.MessageRecordRequest{MessageId: "invalid"})
	if status.Code(err) != codes.InvalidArgument || model.calls != 0 {
		t.Fatalf("invalid input reached storage: %v", err)
	}
}
func TestReadStateRemainsInMessageDomain(t *testing.T) {
	for _, tc := range []struct {
		row  *immodels.ChatLog
		want string
	}{
		{nil, "unknown"}, {&immodels.ChatLog{ChatType: constants.SingleChatType}, "unknown"},
		{&immodels.ChatLog{ChatType: constants.SingleChatType, ReadRecords: []byte{1}}, "known"},
		{&immodels.ChatLog{ChatType: constants.GroupChatType, ReadRecords: []byte{1}}, "approximate"},
	} {
		got, note := deriveReadState(tc.row)
		if got != tc.want || note == "" {
			t.Fatalf("%s %s", got, note)
		}
	}
}
func TestMessageSearchBoundsAndTruncation(t *testing.T) {
	model := &fakeSearchModel{rows: []immodels.MessageReference{{ID: bson.NewObjectID(), SenderID: "u1"}, {ID: bson.NewObjectID(), SenderID: "u1"}}}
	s := NewMessageQueryServer(&svc.ServiceContext{MessageSearch: model})
	resp, err := s.SearchMessages(context.Background(), &im.MessageSearchRequest{SenderId: "u1", StartTime: 1, EndTime: 100, Limit: 1})
	if err != nil || len(resp.Messages) != 1 || !resp.Truncated || model.filter.Limit != 1 {
		t.Fatalf("%+v %v", resp, err)
	}
	for _, req := range []*im.MessageSearchRequest{
		nil, {SenderId: "u1", EndTime: 100}, {SenderId: "u1", StartTime: 100, EndTime: 1},
		{SenderId: "u1", StartTime: 1, EndTime: int64(8 * 24 * time.Hour)},
		{SenderId: "u1", StartTime: 1, EndTime: 100, Limit: 21}, {SenderId: " u1", StartTime: 1, EndTime: 100},
	} {
		if _, err := s.SearchMessages(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("%+v %v", req, err)
		}
	}
	model.err = context.DeadlineExceeded
	if resp, err := s.SearchMessages(context.Background(), &im.MessageSearchRequest{SenderId: "u1", StartTime: 1, EndTime: 100}); resp != nil || status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("%+v %v", resp, err)
	}
}
func TestMessageQueryAuthenticatedGRPC(t *testing.T) {
	model := &fakeRecordModel{err: immodels.ErrNotFound}
	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer(grpc.UnaryInterceptor(serviceauth.Guard(serviceauth.Config{Enable: true, Token: "domain-secret"}, "im.MessageQuery")))
	im.RegisterMessageQueryServer(gs, NewMessageQueryServer(&svc.ServiceContext{ChatLogModel: model}))
	go gs.Serve(lis)
	t.Cleanup(func() { gs.Stop(); lis.Close() })
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := im.NewMessageQueryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &im.MessageRecordRequest{MessageId: bson.NewObjectID().Hex()}
	for _, token := range []string{"", "wrong"} {
		if _, err := client.GetMessageRecord(serviceauth.Outgoing(ctx, token), req); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("token rejected: %v", err)
		}
	}
	if model.calls != 0 {
		t.Fatal("unauthorized request reached storage")
	}
	resp, err := client.GetMessageRecord(serviceauth.Outgoing(ctx, "domain-secret"), req)
	if err != nil || resp.Found || model.calls != 1 {
		t.Fatalf("%+v %v calls=%d", resp, err, model.calls)
	}
}

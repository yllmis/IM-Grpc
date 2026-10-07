package logic

import (
	"context"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/serviceauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

type fakeSearchQuery struct {
	im.MessageQueryClient
	response *im.MessageSearchResponse
	err      error
	request  *im.MessageSearchRequest
	token    string
}

func (f *fakeSearchQuery) SearchMessages(ctx context.Context, in *im.MessageSearchRequest, _ ...grpc.CallOption) (*im.MessageSearchResponse, error) {
	f.request = in
	md, _ := metadata.FromOutgoingContext(ctx)
	f.token = md.Get(serviceauth.MetadataServiceToken)[0]
	return f.response, f.err
}
func TestSearchMessages_ForwardsDomainFactsAndDedicatedCredential(t *testing.T) {
	client := &fakeSearchQuery{response: &im.MessageSearchResponse{
		Messages: []*im.MessageQueryReference{{MessageId: "665f1c0000000000000000aa", SenderId: "u1"}}, Truncated: true, ObservedAt: 42,
	}}
	s := &svc.ServiceContext{MessageQueryRpc: client}
	s.Config.DomainQueryToken = "domain-only-token"
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(serviceauth.MetadataServiceToken, "caller-token"))
	got, err := NewSearchMessagesLogic(ctx, s).SearchMessages(&operations.SearchMessagesRequest{SenderId: "u1", StartTime: 1, EndTime: 100, Limit: 1})
	if err != nil || len(got.Messages) != 1 || !got.Truncated || got.ObservedAt != 42 {
		t.Fatalf("%+v %v", got, err)
	}
	if client.request.SenderId != "u1" || client.request.Limit != 1 || client.token != "domain-only-token" {
		t.Fatalf("forwarding: %+v token=%q", client.request, client.token)
	}
}
func TestSearchMessages_PreservesDomainErrors(t *testing.T) {
	for _, code := range []codes.Code{codes.InvalidArgument, codes.PermissionDenied, codes.Unavailable, codes.DeadlineExceeded, codes.Unimplemented} {
		client := &fakeSearchQuery{err: status.Error(code, "domain query failed")}
		resp, err := NewSearchMessagesLogic(context.Background(), &svc.ServiceContext{MessageQueryRpc: client}).SearchMessages(&operations.SearchMessagesRequest{SenderId: "u1"})
		if status.Code(err) != code || resp != nil {
			t.Fatalf("code=%v resp=%v err=%v", code, resp, err)
		}
	}
}

package logic

import (
	"context"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/pkg/serviceauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestMessageRecord_ObservationFailureDoesNotHideDomainFact(t *testing.T) {
	for _, found := range []bool{true, false} {
		s := &svc.ServiceContext{MessageQueryRpc: &fakeMessageQuery{record: &im.MessageRecordResponse{Found: found, MessageId: "665f1c0000000000000000aa", ObservedAt: 42}}, EventModel: &fakeEventModel{err: context.DeadlineExceeded}}
		resp, err := NewGetMessageRecordLogic(context.Background(), s).GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: "665f1c0000000000000000aa"})
		if err != nil || resp.Found != found || resp.EventsState != "unknown" {
			t.Fatalf("%+v %v", resp, err)
		}
	}
}

type fakeUserQuery struct {
	user.UserQueryClient
	err     error
	request *user.UserReferenceRequest
	token   string
}

func (f *fakeUserQuery) FindUserReference(ctx context.Context, in *user.UserReferenceRequest, _ ...grpc.CallOption) (*user.UserReferenceResponse, error) {
	f.request = in
	md, _ := metadata.FromOutgoingContext(ctx)
	f.token = md.Get(serviceauth.MetadataServiceToken)[0]
	if f.err != nil {
		return nil, f.err
	}
	return &user.UserReferenceResponse{Users: []*user.UserQueryReference{{UserId: "u1", DisplayName: "tester", ObservedAt: 42}}, Truncated: true}, nil
}
func TestUserReference_ForwardsMinimalProjectionAndTruncation(t *testing.T) {
	client := &fakeUserQuery{}
	s := &svc.ServiceContext{UserQueryRpc: client}
	s.Config.DomainQueryToken = "domain-only-token"
	resp, err := NewFindUserReferenceLogic(context.Background(), s).FindUserReference(&operations.FindUserReferenceRequest{UserId: "u1"})
	if err != nil || !resp.Truncated || len(resp.Users) != 1 || resp.Users[0].ObservedAt != 42 {
		t.Fatalf("%+v %v", resp, err)
	}
	if client.request.UserId != "u1" || client.request.Limit != 50 || client.token != "domain-only-token" {
		t.Fatalf("forwarding: %+v token=%q", client.request, client.token)
	}
}
func TestUserReference_PreservesPermissionError(t *testing.T) {
	resp, err := NewFindUserReferenceLogic(context.Background(), &svc.ServiceContext{UserQueryRpc: &fakeUserQuery{err: status.Error(codes.PermissionDenied, "denied")}}).FindUserReference(&operations.FindUserReferenceRequest{UserId: "u1"})
	if resp != nil || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("%+v %v", resp, err)
	}
}
func TestObservationRPCDoesNotNeedDomainClients(t *testing.T) {
	s := &svc.ServiceContext{EventModel: &fakeEventModel{}}
	if _, err := NewGetMessageTimelineLogic(context.Background(), s).GetMessageTimeline(&operations.GetMessageTimelineRequest{MessageId: "665f1c0000000000000000aa"}); err != nil {
		t.Fatal(err)
	}
	resp, _ := NewGetCapabilitiesLogic(context.Background(), s).GetCapabilities(&operations.GetCapabilitiesRequest{})
	if resp.MessageRecord != "unsupported" || resp.MessageSearch != "unsupported" {
		t.Fatalf("%+v", resp)
	}
}

package logic

import (
	"context"
	"testing"

	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeMessageSearchModel struct {
	rows []operationsmodels.MessageReference
	err  error
}

func (f *fakeMessageSearchModel) Search(_ context.Context, _ operationsmodels.MessageSearchFilter) ([]operationsmodels.MessageReference, error) {
	return f.rows, f.err
}

func TestSearchMessages_ReturnsBoundedCandidates(t *testing.T) {
	model := &fakeMessageSearchModel{rows: []operationsmodels.MessageReference{
		{ID: bson.NewObjectID(), SenderID: "sender-1", ReceiverID: "receiver-1", CreatedAt: 10},
		{ID: bson.NewObjectID(), SenderID: "sender-1", ReceiverID: "receiver-2", CreatedAt: 20},
	}}
	logic := NewSearchMessagesLogic(context.Background(), &svc.ServiceContext{MessageSearch: model})
	got, err := logic.SearchMessages(&operations.SearchMessagesRequest{
		SenderId: "sender-1", StartTime: 1, EndTime: 100, Limit: 1,
	})
	if err != nil {
		t.Fatalf("SearchMessages() error = %v", err)
	}
	if len(got.Messages) != 1 || !got.Truncated {
		t.Fatalf("expected one truncated candidate, got %+v", got)
	}
	if got.Messages[0].SenderId != "sender-1" {
		t.Fatalf("sender mismatch: %+v", got.Messages[0])
	}
}

func TestSearchMessages_RejectsUnboundedOrInvalidRange(t *testing.T) {
	logic := NewSearchMessagesLogic(context.Background(), &svc.ServiceContext{
		MessageSearch: &fakeMessageSearchModel{},
	})
	cases := []*operations.SearchMessagesRequest{
		{SenderId: "sender-1", EndTime: 100},
		{SenderId: "sender-1", StartTime: 100, EndTime: 1},
	}
	for _, input := range cases {
		if _, err := logic.SearchMessages(input); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("input %+v code = %v, want InvalidArgument", input, status.Code(err))
		}
	}
}

func TestSearchMessages_DoesNotTurnStorageFailureIntoEmptyResult(t *testing.T) {
	logic := NewSearchMessagesLogic(context.Background(), &svc.ServiceContext{
		MessageSearch: &fakeMessageSearchModel{err: context.DeadlineExceeded},
	})
	_, err := logic.SearchMessages(&operations.SearchMessagesRequest{
		SenderId: "sender-1", StartTime: 1, EndTime: 100,
	})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("code = %v, want DeadlineExceeded", status.Code(err))
	}
}

package logic

import (
	"context"
	"testing"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/internal/svc"
	"github.com/IM_System/pkg/constants"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type putAccessModel struct {
	immodels.ConversationsModel
	data   *immodels.Conversations
	writes int
}

func (f *putAccessModel) FindByUserId(context.Context, string) (*immodels.Conversations, error) {
	return f.data, nil
}
func (f *putAccessModel) Update(context.Context, *immodels.Conversations) (*mongo.UpdateResult, error) {
	f.writes++
	return &mongo.UpdateResult{}, nil
}

func TestPutConversationsCannotChangeMembershipOrType(t *testing.T) {
	for _, tc := range []struct {
		name, key, id string
		kind          int32
		want          codes.Code
	}{
		{"valid", "1_2", "1_2", int32(constants.SingleChatType), codes.OK},
		{"new membership", "3_4", "3_4", int32(constants.SingleChatType), codes.PermissionDenied},
		{"mismatched key", "1_2", "3_4", int32(constants.SingleChatType), codes.PermissionDenied},
		{"changed type", "1_2", "1_2", int32(constants.GroupChatType), codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &putAccessModel{data: &immodels.Conversations{UserId: "1", ConversationList: map[string]*immodels.Conversation{"1_2": {ConversationId: "1_2", ChatType: constants.SingleChatType}}}}
			_, err := NewPutConversationsLogic(context.Background(), &svc.ServiceContext{ConversationsModel: f}).PutConversations(&im.PutConversationsReq{UserId: "1", ConversationList: map[string]*im.Conversation{tc.key: {ConversationId: tc.id, ChatType: tc.kind}}})
			if status.Code(err) != tc.want {
				t.Fatal(err)
			}
			if tc.want != codes.OK && (f.writes != 0 || len(f.data.ConversationList) != 1) {
				t.Fatal("unauthorized mutation occurred")
			}
			if tc.want == codes.OK && f.writes != 1 {
				t.Fatal("normal update rejected")
			}
		})
	}
}

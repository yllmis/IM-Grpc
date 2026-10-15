package conversation

import (
	"context"
	"testing"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/pkg/constants"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type readAccessModel struct {
	immodels.ChatLogModel
	rows  []*immodels.ChatLog
	calls int
}

func (f *readAccessModel) ListByIds(context.Context, []string) ([]*immodels.ChatLog, error) {
	f.calls++
	return f.rows, nil
}

type groupAccessClient struct {
	socialclient.Social
	members []*social.GroupMembers
	err     error
}

func (f *groupAccessClient) Groupusers(context.Context, *social.GroupusersReq, ...grpc.CallOption) (*social.GroupusersResp, error) {
	return &social.GroupusersResp{List: f.members}, f.err
}

func TestWSRejectsForgedConversationAndNonmemberBeforeKafka(t *testing.T) {
	s := &svc.ServiceContext{Social: &groupAccessClient{members: []*social.GroupMembers{{UserId: "2"}}}}
	_, err := authorizeChat(context.Background(), s, "1", "2", "3_4", constants.SingleChatType)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatal("forged conversation accepted")
	}
	_, err = authorizeChat(context.Background(), s, "1", "g1", "g1", constants.GroupChatType)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatal("nonmember group send accepted")
	}
	id, err := authorizeChat(context.Background(), s, "1", "2", "", constants.SingleChatType)
	if err != nil || id != "1_2" {
		t.Fatal("normal private chat broken")
	}
}

func TestWSReadBatchBelongsToRecipientAndConversation(t *testing.T) {
	oid := bson.NewObjectID()
	for _, tc := range []struct {
		name, conv, recipient string
		want                  codes.Code
	}{
		{"valid", "1_2", "1", codes.OK}, {"other conversation", "3_4", "1", codes.PermissionDenied},
		{"sender cannot mark recipient read", "1_2", "2", codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &readAccessModel{rows: []*immodels.ChatLog{{ID: oid, ConversationId: tc.conv, RecvId: tc.recipient, ChatType: constants.SingleChatType}}}
			_, err := authorizeRead(context.Background(), &svc.ServiceContext{ChatLogModel: f}, "1", "2", "1_2", constants.SingleChatType, []string{oid.Hex()})
			if status.Code(err) != tc.want {
				t.Fatal(err)
			}
		})
	}
	f := &readAccessModel{}
	for _, ids := range [][]string{nil, {"invalid"}, make([]string, 101)} {
		_, err := authorizeRead(context.Background(), &svc.ServiceContext{ChatLogModel: f}, "1", "2", "", constants.SingleChatType, ids)
		if status.Code(err) != codes.InvalidArgument || f.calls != 0 {
			t.Fatal("invalid IDs reached storage")
		}
	}
	_, err := authorizeRead(context.Background(), &svc.ServiceContext{ChatLogModel: f}, "1", "2", "", constants.SingleChatType, []string{oid.Hex()})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatal("missing message accepted")
	}
}

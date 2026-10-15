package logic

import (
	"context"
	"errors"
	"testing"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/api/internal/types"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/apps/im/rpc/imclient"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/user/rpc/user"
	"github.com/IM_System/apps/user/rpc/userclient"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accessIm struct {
	imclient.Im
	conversations                   map[string]*im.Conversation
	chat                            *im.ChatLog
	lookupErr                       error
	chatCalls, setupCalls, putCalls int
	setupSender                     string
}

func (f *accessIm) GetConversations(_ context.Context, in *im.GetConversationsReq, _ ...grpc.CallOption) (*im.GetConversationsResp, error) {
	if in.UserId != "1" {
		return nil, errors.New("wrong authenticated user")
	}
	return &im.GetConversationsResp{ConversationList: f.conversations}, f.lookupErr
}
func (f *accessIm) GetChatLog(context.Context, *im.GetChatLogReq, ...grpc.CallOption) (*im.GetChatLogResp, error) {
	f.chatCalls++
	return &im.GetChatLogResp{List: []*im.ChatLog{f.chat}}, nil
}
func (f *accessIm) SetUpUserConversation(_ context.Context, in *im.SetUpUserConversationReq, _ ...grpc.CallOption) (*im.SetUpUserConversationResp, error) {
	f.setupCalls++
	f.setupSender = in.SendId
	return &im.SetUpUserConversationResp{}, nil
}
func (f *accessIm) PutConversations(context.Context, *im.PutConversationsReq, ...grpc.CallOption) (*im.PutConversationsResp, error) {
	f.putCalls++
	return &im.PutConversationsResp{}, nil
}

type accessSocial struct {
	socialclient.Social
	members []*social.GroupMembers
	err     error
}

func (f *accessSocial) Groupusers(context.Context, *social.GroupusersReq, ...grpc.CallOption) (*social.GroupusersResp, error) {
	return &social.GroupusersResp{List: f.members}, f.err
}

type accessUser struct{ userclient.User }

func (*accessUser) FindUser(context.Context, *user.FindUserReq, ...grpc.CallOption) (*user.FindUserResp, error) {
	return &user.FindUserResp{}, nil
}

func userContext() context.Context {
	return context.WithValue(context.Background(), ctxdata.Identify, "1")
}

func TestChatHistoryRejectsForeignConversationBeforeReadingMessages(t *testing.T) {
	for _, tc := range []struct {
		name, id  string
		lookupErr error
		want      codes.Code
	}{
		{"own", "1_2", nil, codes.OK}, {"foreign", "3_4", nil, codes.PermissionDenied},
		{"storage failure", "1_2", status.Error(codes.Unavailable, "storage offline"), codes.Unavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &accessIm{conversations: map[string]*im.Conversation{"1_2": {ConversationId: "1_2", ChatType: int32(constants.SingleChatType)}}, chat: &im.ChatLog{Id: "message"}, lookupErr: tc.lookupErr}
			resp, err := NewGetChatLogLogic(userContext(), &svc.ServiceContext{Im: f}).GetChatLog(&types.ChatLogReq{ConversationId: tc.id})
			if status.Code(err) != tc.want {
				t.Fatal(err)
			}
			if tc.want != codes.OK && (resp != nil || f.chatCalls != 0) {
				t.Fatal("foreign messages read")
			}
			if tc.want == codes.OK && f.chatCalls != 1 {
				t.Fatal("own history blocked")
			}
		})
	}
	f := &accessIm{}
	_, err := NewGetChatLogLogic(context.Background(), &svc.ServiceContext{Im: f}).GetChatLog(&types.ChatLogReq{ConversationId: "1_2"})
	if status.Code(err) != codes.Unauthenticated || f.chatCalls != 0 {
		t.Fatal("missing user accepted")
	}
}

func TestGroupHistoryRequiresCurrentMembership(t *testing.T) {
	f := &accessIm{conversations: map[string]*im.Conversation{"g1": {ConversationId: "g1", ChatType: int32(constants.GroupChatType)}}, chat: &im.ChatLog{}}
	socialRPC := &accessSocial{members: []*social.GroupMembers{{UserId: "2"}}}
	s := &svc.ServiceContext{Im: f, Social: socialRPC}
	_, err := NewGetChatLogLogic(userContext(), s).GetChatLog(&types.ChatLogReq{ConversationId: "g1"})
	if status.Code(err) != codes.PermissionDenied || f.chatCalls != 0 {
		t.Fatal("stale group conversation granted membership")
	}
	socialRPC.members = []*social.GroupMembers{{UserId: "1"}}
	if _, err := NewGetChatLogLogic(userContext(), s).GetChatLog(&types.ChatLogReq{ConversationId: "g1"}); err != nil {
		t.Fatal(err)
	}
}

func TestSetUpConversationUsesAuthenticatedSender(t *testing.T) {
	f := &accessIm{}
	s := &svc.ServiceContext{Im: f, Social: &accessSocial{}}
	l := NewSetUpUserConversationLogic(userContext(), s)
	_, err := l.SetUpUserConversation(&types.SetUpUserConversationReq{SendId: "2", RecvId: "3", ChatType: int32(constants.SingleChatType)})
	if status.Code(err) != codes.PermissionDenied || f.setupCalls != 0 {
		t.Fatal("forged sender accepted")
	}
	_, err = l.SetUpUserConversation(&types.SetUpUserConversationReq{RecvId: "2", ChatType: int32(constants.SingleChatType)})
	if err != nil || f.setupSender != "1" {
		t.Fatal("sender not derived from JWT")
	}
	_, err = l.SetUpUserConversation(&types.SetUpUserConversationReq{RecvId: "g1", ChatType: int32(constants.GroupChatType)})
	if status.Code(err) != codes.PermissionDenied || f.setupCalls != 1 {
		t.Fatal("nonmember created group conversation")
	}
}

func TestReadRecordsRejectThirdParty(t *testing.T) {
	f := &accessIm{chat: &im.ChatLog{SendId: "2", RecvId: "3", ChatType: int32(constants.SingleChatType)}}
	s := &svc.ServiceContext{Im: f, User: &accessUser{}}
	resp, err := NewGetChatLogReadRecordsLogic(userContext(), s).GetChatLogReadRecords(&types.GetChatLogReadRecordsReq{MsgId: "message"})
	if resp != nil || status.Code(err) != codes.PermissionDenied {
		t.Fatal("third party read state exposed")
	}
	f.chat.SendId = "1"
	if _, err := NewGetChatLogReadRecordsLogic(userContext(), s).GetChatLogReadRecords(&types.GetChatLogReadRecordsReq{MsgId: "message"}); err != nil {
		t.Fatal(err)
	}
}

func TestPutConversationDoesNotGrantForeignMembership(t *testing.T) {
	f := &accessIm{}
	resp, err := NewPutConversationsLogic(userContext(), &svc.ServiceContext{Im: f}).PutConversations(&types.PutConversationsReq{ConversationList: map[string]*types.Conversation{"3_4": {ConversationId: "3_4"}}})
	if resp != nil || status.Code(err) != codes.PermissionDenied || f.putCalls != 0 {
		t.Fatal("foreign conversation injected")
	}
}

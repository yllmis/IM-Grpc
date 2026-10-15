package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IM_System/apps/im/ws/internal/svc"
	imws "github.com/IM_System/apps/im/ws/websocket"
	"github.com/IM_System/apps/social/rpc/social"
	"github.com/IM_System/apps/social/rpc/socialclient"
	"github.com/IM_System/apps/task/mq/mq"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

type permissionChatProducer struct{ messages chan *mq.MsgChatTransfer }

func (p *permissionChatProducer) Push(message *mq.MsgChatTransfer) error {
	p.messages <- message
	return nil
}

type permissionGroupRPC struct{ socialclient.Social }

func (*permissionGroupRPC) Groupusers(context.Context, *social.GroupusersReq, ...grpc.CallOption) (*social.GroupusersResp, error) {
	return &social.GroupusersResp{List: []*social.GroupMembers{{UserId: "2"}}}, nil
}

func TestWSMessagePermissionsBeforeProducer(t *testing.T) {
	producer := &permissionChatProducer{messages: make(chan *mq.MsgChatTransfer, 3)}
	services := &svc.ServiceContext{MsgChatTransferClient: producer, Social: &permissionGroupRPC{}}
	services.Config.JwtAuth.AccessSecret = "ws-chat-test-key-only"
	server := imws.NewServer("", imws.WithAuthentication(NewJwtAuth(services)))
	RegisterHandlers(server, services)
	httpServer := httptest.NewServer(http.HandlerFunc(server.ServerWs))
	defer httpServer.Close()
	token, err := ctxdata.GetTokenKey(services.Config.JwtAuth.AccessSecret, time.Now().Unix(), 60, "1")
	if err != nil {
		t.Fatal(err)
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), header)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, data := range []map[string]any{
		{"chatType": int(constants.SingleChatType), "recvId": "2", "conversationId": "3_4"},
		{"chatType": int(constants.GroupChatType), "recvId": "g1", "conversationId": "g1"},
	} {
		if err := conn.WriteJSON(&imws.Message{Method: "conversation.chat", Data: data}); err != nil {
			t.Fatal(err)
		}
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var response imws.Message
		if err := conn.ReadJSON(&response); err != nil {
			t.Fatal(err)
		}
		if response.FrameType != imws.FrameErr || len(producer.messages) != 0 {
			t.Fatal("unauthorized message reached Kafka producer")
		}
	}
	if err := conn.WriteJSON(&imws.Message{Id: "client-message", Method: "conversation.chat", FormId: "forged-user", Data: map[string]any{"chatType": int(constants.SingleChatType), "recvId": "2", "content": "test"}}); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var response imws.Message
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response.FrameType != imws.FrameData {
		t.Fatal("normal chat rejected")
	}
	select {
	case message := <-producer.messages:
		if message.SendId != "1" || message.ConversationId != "1_2" {
			t.Fatal("sender did not come from authenticated connection")
		}
	default:
		t.Fatal("normal chat did not reach producer")
	}
}

func TestAuthenticatedWSInternalRoutes(t *testing.T) {
	services := &svc.ServiceContext{}
	services.Config.JwtAuth.AccessSecret = "ws-test-signing-key-only"
	server := imws.NewServer("", imws.WithAuthentication(NewJwtAuth(services)))
	RegisterHandlers(server, services)
	httpServer := httptest.NewServer(http.HandlerFunc(server.ServerWs))
	t.Cleanup(httpServer.Close)
	dial := func(uid string) *websocket.Conn {
		token, err := ctxdata.GetTokenKey(services.Config.JwtAuth.AccessSecret, time.Now().Unix(), 60, uid)
		if err != nil {
			t.Fatal(err)
		}
		header := http.Header{}
		header.Set("Authorization", "Bearer "+token)
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), header)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	ordinary := dial("1")
	for _, method := range []string{"push", "user.online"} {
		if err := ordinary.WriteJSON(&imws.Message{Method: method, FormId: constants.SYSTEM_ROOT_ID, Data: map[string]any{"sendId": constants.SYSTEM_ROOT_ID}}); err != nil {
			t.Fatal(err)
		}
		ordinary.SetReadDeadline(time.Now().Add(3 * time.Second))
		var response imws.Message
		if err := ordinary.ReadJSON(&response); err != nil {
			t.Fatal(err)
		}
		if response.FrameType != imws.FrameErr {
			t.Fatal("ordinary user reached internal WS route")
		}
	}
	system := dial(constants.SYSTEM_ROOT_ID)
	// An offline-recipient push succeeds without a response; the subsequent
	// online query is a barrier proving the system push was not rejected.
	if err := system.WriteJSON(&imws.Message{Method: "push", Data: map[string]any{"chatType": int(constants.SingleChatType), "recvId": "offline-test-user", "sendId": "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := system.WriteJSON(&imws.Message{Method: "user.online"}); err != nil {
		t.Fatal(err)
	}
	system.SetReadDeadline(time.Now().Add(3 * time.Second))
	var response imws.Message
	if err := system.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response.FrameType != imws.FrameData || response.FormId != constants.SYSTEM_ROOT_ID {
		t.Fatal("authenticated system route broken")
	}
}

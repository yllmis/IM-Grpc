package logic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IM_System/apps/im/api/internal/svc"
	"github.com/IM_System/apps/im/api/internal/types"
	"github.com/IM_System/apps/im/rpc/im"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/ctxdata"
	"github.com/IM_System/pkg/resultx"
	"github.com/zeromicro/go-zero/rest/handler"
)

func TestHTTPJWTAndConversationAuthorization(t *testing.T) {
	const secret = "http-test-signing-key-only"
	token, err := ctxdata.GetTokenKey(secret, time.Now().Unix(), 60, "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, credential, conversation string
		want                           int
	}{
		{"missing JWT", "", "1_2", http.StatusUnauthorized},
		{"service token is not a user JWT", "read-query-token-only", "1_2", http.StatusUnauthorized},
		{"own conversation", token, "1_2", http.StatusOK},
		{"foreign conversation", token, "3_4", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &accessIm{conversations: map[string]*im.Conversation{"1_2": {ConversationId: "1_2", ChatType: int32(constants.SingleChatType)}}, chat: &im.ChatLog{}}
			h := handler.Authorize(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := NewGetChatLogLogic(r.Context(), &svc.ServiceContext{Im: f}).GetChatLog(&types.ChatLogReq{ConversationId: tc.conversation})
				if err != nil {
					code, _ := resultx.ErrHandler("test")(r.Context(), err)
					w.WriteHeader(code)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/v1/im/chatlog", nil)
			if tc.credential != "" {
				req.Header.Set("Authorization", "Bearer "+tc.credential)
			}
			response := httptest.NewRecorder()
			h.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d", response.Code, tc.want)
			}
			if tc.want != http.StatusOK && f.chatCalls != 0 {
				t.Fatal("denied request read messages")
			}
		})
	}
}

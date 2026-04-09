package handler

import (
	"github.com/IM_System/apps/im/ws/internal/handler/conversation"
	"github.com/IM_System/apps/im/ws/internal/handler/push"
	"github.com/IM_System/apps/im/ws/internal/handler/user"
	"github.com/IM_System/apps/im/ws/internal/svc"
	"github.com/IM_System/apps/im/ws/websocket"
)

func RegisterHandlers(srv *websocket.Server, svc *svc.ServiceContext) {
	srv.AddRoutes([]websocket.Route{
		{
			Method:  "user.online",
			Handler: user.Online(svc),
		},
		{
			Method:  "conversation.chat",
			Handler: conversation.Chat(svc),
		},
		{
			Method:  "push",
			Handler: push.Push(svc),
		},
	})
}

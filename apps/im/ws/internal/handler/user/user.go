package user

import (
	"github.com/IM_System/apps/im/ws/internal/svc"
	websocketx "github.com/IM_System/apps/im/ws/websocket"
	"github.com/gorilla/websocket"
)

// 获取所有在线用户
func Online(svc *svc.ServiceContext) websocketx.HandlerFunc {
	return func(srv *websocketx.Server, conn *websocket.Conn, msg *websocketx.Message) {
		// 获取在线用户列表
		uids := srv.GetUsers()
		u := srv.GetUsers(conn)
		err := srv.Send(websocketx.NewMessage(u[0], uids), conn)
		srv.Info("err", err)

	}

}

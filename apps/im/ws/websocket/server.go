package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type Server struct {
	sync.RWMutex

	authentication Authentication
	routes         map[string]HandlerFunc
	addr           string
	patten         string
	opt            *serverOption

	// 连接管理
	connToUser map[*Conn]string
	userToConn map[string]*Conn

	upgrader websocket.Upgrader
	logx.Logger
}

// 初始化服务
func NewServer(addr string, opts ...ServerOptions) *Server {
	opt := newServerOptions(opts...)

	return &Server{
		routes:   make(map[string]HandlerFunc),
		addr:     addr,
		patten:   opt.patten,
		upgrader: websocket.Upgrader{},
		opt:      &opt,

		authentication: opt.Authentication,

		connToUser: make(map[*Conn]string),
		userToConn: make(map[string]*Conn),

		Logger: logx.WithContext(context.Background()),
	}
}

// 接收请求，处理请求
func (s *Server) ServerWs(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.Errorf("server handler ws recover err %v", r)
		}
	}()

	conn := NewConn(s, w, r)
	if conn == nil {
		return
	}
	// conn, err := s.upgrader.Upgrade(w, r, nil)
	// if err != nil {
	// 	s.Errorf("upgrade err %v", err)
	// 	return
	// }

	if !s.authentication.Auth(w, r) {
		s.Send(&Message{
			FrameType: FrameData,
			Data:      fmt.Sprintf("不具备访问权限"),
		}, conn)
		// conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不具备访问权限")))
		conn.Close()
		return
	}

	// 添加连接
	s.addConn(conn, r)

	// 处理连接
	go s.handleConn(conn)

}

// 处理请求
func (s *Server) handleConn(conn *Conn) {
	for {
		// 获取请求消息
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.Errorf("websocket conn read message err %v", err)
			s.Close(conn)
			return
		}

		// 解析消息
		var message Message
		if err = json.Unmarshal(msg, &message); err != nil {
			s.Errorf("websocket unmarshal message err %v, msg %s", err, msg)
			s.Close(conn)
			return
		}

		switch message.FrameType {
		case FramePing:
			s.Send(&Message{
				FrameType: FramePing}, conn)
		case FrameData:
			// 根据请求的method分发对应的路由
			if handler, ok := s.routes[message.Method]; ok {
				handler(s, conn, &message)
			} else {
				s.Send(&Message{
					FrameType: FrameData,
					Data:      fmt.Sprintf("不存在执行的方法: %s，请检查", message.Method),
				}, conn)

				// conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不存在执行的方法: %s，请检查", message.Method)))
			}

		}

	}
}

func (s *Server) addConn(conn *Conn, req *http.Request) {
	uid := s.authentication.UserId(req)

	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	// 如果用户已经存在，则关闭之前的连接
	if c := s.userToConn[uid]; c != nil {
		s.Close(c)
	}

	s.connToUser[conn] = uid
	s.userToConn[uid] = conn
}

// 根据用户id获取连接
func (s *Server) GetConn(uid string) *Conn {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.userToConn[uid]
}

func (s *Server) GetConns(uids ...string) []*Conn {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	var conns []*Conn
	for _, uid := range uids {
		if conn, ok := s.userToConn[uid]; ok {
			conns = append(conns, conn)
		}
	}

	return conns
}

func (s *Server) SendByUserId(msg interface{}, sendIds ...string) error {
	if len(sendIds) == 0 {
		return nil
	}

	return s.Send(msg, s.GetConns(sendIds...)...)
}

func (s *Server) Send(msg interface{}, conns ...*Conn) error {
	if len(conns) == 0 {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
	}
	return nil
}

// 根据连接获取用户
func (s *Server) GetUser(conn *Conn) string {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.connToUser[conn]
}

func (s *Server) GetUsers(conns ...*Conn) []string {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	var users []string
	if len(conns) == 0 {
		// 获取全部
		users = make([]string, 0, len(s.connToUser))
		for _, uid := range s.connToUser {
			users = append(users, uid)
		}
	} else {
		// 获取部分
		users = make([]string, 0, len(conns))
		for _, conn := range conns {
			users = append(users, s.connToUser[conn])
		}
	}
	return users
}

func (s *Server) Close(conn *Conn) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	uid := s.connToUser[conn]
	if uid == "" {
		// 已经被关闭
		return
	}

	delete(s.connToUser, conn)
	delete(s.userToConn, uid)

	conn.Close()
}

// 添加路由
func (s *Server) AddRoutes(rs []Route) {
	for _, r := range rs {
		s.routes[r.Method] = r.Handler
	}
}

// 启动服务
func (s *Server) Start() {
	http.HandleFunc(s.patten, s.ServerWs)
	s.Info(http.ListenAndServe(s.addr, nil))
}

// 停止服务
func (s *Server) Stop() {
	fmt.Println("WebSocket server stopped")
}

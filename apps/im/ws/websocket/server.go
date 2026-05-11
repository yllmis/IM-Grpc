package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type AckType int

const (
	NoAck    AckType = iota // 不需要ack确认
	OnlyAck                 // 只需要ack确认，不需要重传，直接从消息队列中删除
	RigorAck                // 需要ack确认，且需要重传，直到收到ack确认才从消息队列中删除
)

func (t AckType) ToString() string {
	switch t {
	case OnlyAck:
		return "OnlyAck"
	case RigorAck:
		return "RigorAck"
	}
	return "NoAck"
}

type Server struct {
	sync.RWMutex

	authentication Authentication
	routes         map[string]HandlerFunc
	addr           string
	patten         string
	opt            *serverOption

	// 并发处理
	*threading.TaskRunner

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

		Logger:     logx.WithContext(context.Background()),
		TaskRunner: threading.NewTaskRunner(opt.concurrency),
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

	uids := s.GetUsers(conn)
	conn.Uid = uids[0]

	go s.handleWrite(conn)

	if s.isAck(nil) {
		go s.readAck(conn)
	}

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
			s.Errorf("websocket unmarshal message err %v, msg %s", err, string(msg))
			s.Close(conn)
			return
		}

		// todo: 给客户端回复ack消息，告知消息已被处理

		// 根据消息处理请求
		if s.isAck(&message) {
			// 需要ack确认的消息，放入消息队列，等待ack确认
			s.Infof("conn message read ack msg %v", message)
			conn.appendMsgMq(&message)
		} else {
			conn.message <- &message
		}
	}
}

func (s *Server) isAck(message *Message) bool {

	if message == nil {
		return s.opt.ack != NoAck
	}

	return s.opt.ack != NoAck && message.FrameType != FrameAck
}

// 读取的ack确认
func (s *Server) readAck(conn *Conn) {

	send := func(msg *Message, conn *Conn) error {
		err := s.Send(msg, conn)
		if err == nil {
			return nil
		}

		s.Errorf("message ack OnlyAck send err %v messgae %v", err, msg)
		conn.readMessage[0].errCount++
		conn.MessageMu.Unlock()

		tempDelay := time.Duration(200*conn.readMessage[0].errCount) * time.Microsecond // 线性退避
		if max := 1 * time.Second; tempDelay > max {                                    // 最大退避时间为1秒
			tempDelay = max
		}

		time.Sleep(tempDelay)
		return err
	}

	for {
		select {
		case <-conn.done:
			// 连接已关闭，退出处理
			s.Infof("close message ack uid %v", conn.Uid)
			return
		default:
		}

		conn.MessageMu.Lock()
		if len(conn.readMessage) == 0 {
			conn.MessageMu.Unlock()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 取出消息队列中的第一条消息，等待ack确认
		message := conn.readMessage[0]
		if message.errCount > s.opt.sendErrCount {
			// 超过最大重试次数，放弃发送
			s.Infof("conn send fail,message %v,ackType %v, maxSendErrCount %v", message, s.opt.ack.ToString(), s.opt.sendErrCount)
			conn.MessageMu.Unlock()
			delete(conn.readMessageReq, message.Id)
			conn.readMessage = conn.readMessage[1:]
			continue
		}

		// 判断ack的方式
		switch s.opt.ack {
		case OnlyAck:
			// 直接给客户端发送消息
			if err := send(&Message{
				FrameType: FrameAck,
				Id:        message.Id,
				AckSeq:    message.AckSeq + 1,
			}, conn); err != nil {
				continue
			}
			// 只需要ack确认，不需要重传，直接从消息队列中删除
			conn.readMessage = conn.readMessage[1:]
			conn.MessageMu.Unlock()

			conn.message <- message
		case RigorAck:
			// 先回
			if message.AckSeq == 0 {
				// 还未确认
				conn.readMessage[0].AckSeq++
				conn.readMessage[0].ackTime = time.Now()
				if err := send(&Message{
					FrameType: FrameAck,
					Id:        message.Id,
					AckSeq:    message.AckSeq,
				}, conn); err != nil {
					continue
				}
				s.Infof("message ack RigorAck send mid %v, seq %v, time %v", message.Id, message.AckSeq, message.ackTime)
				conn.MessageMu.Unlock()
				continue
			}

			// 再验证
			// 1.客户端返回结果，再一次确认
			// 得到客户端的序号
			msgSeq := conn.readMessageReq[message.Id]
			if msgSeq.AckSeq > message.AckSeq {
				// 已经收到过ack确认了，说明消息已经被处理了，直接从消息队列中删除
				conn.readMessage = conn.readMessage[1:]
				conn.MessageMu.Unlock()
				conn.message <- message
				s.Infof("message ack RigorAck success mid %v, seq %v", message.Id, message.AckSeq)
				continue
			}

			// 2.客户端没有确认，是否超时
			val := s.opt.ackTimeout - time.Since(message.ackTime)
			if !message.ackTime.IsZero() && val <= 0 {
				// 2.1 超时
				delete(conn.readMessageReq, message.Id)
				conn.readMessage = conn.readMessage[1:]
				conn.MessageMu.Unlock()
				s.Infof("message ack RigorAck timeout mid %v, seq %v", message.Id, message.AckSeq)
				continue
			}
			// 2.2 未超时，重新发送
			conn.MessageMu.Unlock()
			if val > 0 && val > 300*time.Microsecond {
				if err := send(&Message{
					FrameType: FrameAck,
					Id:        message.Id,
					AckSeq:    message.AckSeq,
				}, conn); err != nil {
					continue
				}
			}
			// 睡眠一定的时间
			time.Sleep(300 * time.Microsecond)
		}

	}

}

// 任务的处理
func (s *Server) handleWrite(conn *Conn) {
	for {
		select {
		case <-conn.done:
			// 连接已关闭，退出处理
			return
		case message := <-conn.message:
			// 根据消息类型处理请求
			switch message.FrameType {
			case FramePing:
				s.Send(&Message{
					FrameType: FramePing}, conn)
			case FrameData:
				// 根据请求的method分发对应的路由
				if handler, ok := s.routes[message.Method]; ok {
					handler(s, conn, message)
				} else {
					s.Send(&Message{
						FrameType: FrameData,
						Data:      fmt.Sprintf("不存在执行的方法: %s，请检查", message.Method),
					}, conn)

					// conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不存在执行的方法: %s，请检查", message.Method)))
				}

			}

			if s.isAck(message) {
				// 删除消息记录
				conn.MessageMu.Lock()
				delete(conn.readMessageReq, message.Id)
				conn.MessageMu.Unlock()
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

	if s.opt.onClose != nil {
		s.opt.onClose(uid)
	}
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

package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	idleMu sync.Mutex

	*websocket.Conn
	s *Server

	idle              time.Time
	maxConnectionIdle time.Duration

	done chan struct{}
}

func NewConn(s *Server, w http.ResponseWriter, r *http.Request) *Conn {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Errorf("upgrade err %v", err)
		return nil
	}

	conn := &Conn{
		Conn:              c,
		s:                 s,
		idle:              time.Now(),
		maxConnectionIdle: s.opt.maxIdleConnection,
		done:              make(chan struct{}),
	}

	go conn.keepAlive()
	return conn
}

func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {

	messageType, p, err = c.Conn.ReadMessage()
	c.idleMu.Lock()
	defer c.idleMu.Unlock()
	if err != nil {
		return
	}

	c.idle = time.Now()

	return
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.idleMu.Lock()
	defer c.idleMu.Unlock()
	err := c.Conn.WriteMessage(messageType, data)
	if err != nil {
		return err
	}

	c.idle = time.Now()

	return nil
}

func (c *Conn) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return c.Conn.Close()
}

// 心跳检测，超过最大空闲时间则关闭连接  借鉴grpc的keepalive机制
func (c *Conn) keepAlive() {
	idleTimer := time.NewTimer(c.maxConnectionIdle)

	defer func() {
		// We need to drain the underlying channel in these timers after a call
		// to Stop(), only if we are interested in resetting them. Clearly we
		// are not interested in resetting them here.
		idleTimer.Stop()
	}()

	for {
		select {
		case <-idleTimer.C:
			c.idleMu.Lock()
			idle := c.idle
			if idle.IsZero() { // The connection is non-idle.
				c.idleMu.Unlock()
				idleTimer.Reset(c.maxConnectionIdle)
				continue
			}
			val := c.maxConnectionIdle - time.Since(idle)
			c.idleMu.Unlock()
			if val <= 0 {
				// The connection has been idle for a duration of keepalive.MaxConnectionIdle or more.
				// Gracefully close the connection.
				c.s.Close(c)
				return
			}
			idleTimer.Reset(val)
		case <-c.done:
			return
		}
	}

}

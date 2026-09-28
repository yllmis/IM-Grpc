package websocket

import "go.mongodb.org/mongo-driver/v2/bson"

func newConnID() string {
	return bson.NewObjectID().Hex()
}

// Observer 连接与传输层 ACK 的旁路观察接口。
// 仅供观测记录，不得反向影响连接表、重试与回包语义。
type Observer interface {
	// OnConnect 连接进入连接表（鉴权通过后）
	OnConnect(uid, connectionID string)
	// OnDisconnect 连接关闭；reason: connect-replaced | timeout | disconnect | read-error
	OnDisconnect(uid, connectionID, reason string)
	// OnAckReceived 收到真实客户端 ACK（仅 RigorAck）
	OnAckReceived(uid, frameID string, ackSeq int)
	// OnAckTimeout RigorAck 真实超时
	OnAckTimeout(uid, frameID string, ackSeq int)
}

// nopObserver 默认空实现。
type nopObserver struct{}

func (nopObserver) OnConnect(string, string)            {}
func (nopObserver) OnDisconnect(string, string, string) {}
func (nopObserver) OnAckReceived(string, string, int)   {}
func (nopObserver) OnAckTimeout(string, string, int)    {}

func defaultObserver() Observer { return nopObserver{} }

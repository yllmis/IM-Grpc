package websocket

import "time"

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication

	ack        AckType
	ackTimeout time.Duration

	// ackObserve 仅在 RigorAck 且显式开启观测时记 ack_*；NoAck/OnlyAck 禁止伪造
	ackObserve bool
	observer   Observer

	sendErrCount int // 消息发送失败重试次数，超过这个次数就放弃发送

	patten string

	maxIdleConnection time.Duration

	concurrency int

	onClose func(uid string)
}

func newServerOptions(opts ...ServerOptions) serverOption {
	o := serverOption{
		Authentication:    new(authentication),
		maxIdleConnection: defaultMaxConnectionIdle,
		ackTimeout:        defaultAckTimeout,
		sendErrCount:      defaultSendErrCount,
		patten:            "/ws",
		concurrency:       defaultConcurrency,
		observer:          defaultObserver(),
	}

	// 依次调用每个 option 函数来修改 serverOption 的值
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func WithAuthentication(auth Authentication) ServerOptions {
	return func(opt *serverOption) {
		opt.Authentication = auth
	}
}

func WithServePatten(patten string) ServerOptions {
	return func(opt *serverOption) {
		opt.patten = patten
	}
}

func WithAck(ack AckType) ServerOptions {
	return func(opt *serverOption) {
		opt.ack = ack
	}
}

func WithMaxIdleConnectionIdle(maxIdleConnectionTime time.Duration) ServerOptions {
	return func(opt *serverOption) {
		if maxIdleConnectionTime > 0 {
			opt.maxIdleConnection = maxIdleConnectionTime
		}
	}
}

func WithSendErrCount(count int) ServerOptions {
	return func(opt *serverOption) {
		if count > 0 {
			opt.sendErrCount = count
		}
	}
}

func WithOnClose(fn func(uid string)) ServerOptions {
	return func(opt *serverOption) {
		opt.onClose = fn
	}
}

// WithObserver 注入连接/ACK 观察者；nil 则忽略。
func WithObserver(o Observer) ServerOptions {
	return func(opt *serverOption) {
		if o != nil {
			opt.observer = o
		}
	}
}

// WithAckObserve 开启传输层 ACK 事件观测。
// 仅对 RigorAck 有意义；NoAck/OnlyAck 下调用也不会写 ack_*（见 readAck）。
func WithAckObserve(enabled bool) ServerOptions {
	return func(opt *serverOption) {
		opt.ackObserve = enabled
	}
}

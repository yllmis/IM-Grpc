package websocket

import "time"

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication

	ack        AckType
	ackTimeout time.Duration

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

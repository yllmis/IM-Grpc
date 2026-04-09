package websocket

import "time"

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication
	patten string

	maxIdleConnection time.Duration
}

func newServerOptions(opts ...ServerOptions) serverOption {
	o := serverOption{
		Authentication: new(authentication),
		patten:         "/ws",
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

func WithMaxIdleConnectionIdle(maxIdleConnectionTime time.Duration) ServerOptions {
	return func(opt *serverOption) {
		if maxIdleConnectionTime > 0 {
			opt.maxIdleConnection = maxIdleConnectionTime
		}
	}
}

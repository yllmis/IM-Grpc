package websocket

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication
	patten string
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

func WithServePattern(patten string) ServerOptions {
	return func(opt *serverOption) {
		opt.patten = patten
	}
}

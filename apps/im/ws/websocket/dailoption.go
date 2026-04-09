package websocket

import "net/http"

type DailOptions func(opt *dailOption)

type dailOption struct {
	patten string      // 连接的 websocket 地址
	header http.Header // 连接时携带的 header 信息
}

func newDailOptions(opts ...DailOptions) dailOption {
	o := dailOption{
		patten: "/ws",
		header: http.Header{},
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func WithClientPatten(patten string) DailOptions {
	return func(opt *dailOption) {
		opt.patten = patten
	}
}

func WithClientHeader(header http.Header) DailOptions {
	return func(opt *dailOption) {
		opt.header = header
	}
}

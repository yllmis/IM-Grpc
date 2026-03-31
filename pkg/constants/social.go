package constants

// 处理结果 1.未处理 2.通过 3.拒绝 4.撤销
type HandlerResult int

const (
	NoHandlerResult HandlerResult = iota + 1
	PassHandlerResult
	RefuseHandlerResult
	CancelHandlerResult
)

package constants

type MType int

const (
	MTypeText MType = iota
)

type ChatType int

const (
	GroupChatType ChatType = iota + 1
	SingleChatType
)

type ContentType int

const (
	ContentChatType ContentType = iota // 消息内容
	ContentReadType                    // 已读回执
)

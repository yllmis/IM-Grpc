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

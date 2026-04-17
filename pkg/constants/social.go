package constants

// 处理结果 1.未处理 2.通过 3.拒绝 4.撤销
type HandlerResult int

const (
	NoHandlerResult HandlerResult = iota + 1
	PassHandlerResult
	RefuseHandlerResult
	CancelHandlerResult
)

// 群等级 1. 创建者，2. 管理者，3. 普通
type GroupRoleLevel int

const (
	CreatorGroupRoleLevel GroupRoleLevel = iota + 1 // 为什么会 从1开始？
	ManagerGroupRoleLevel
	AtLargeGroupRoleLevel
)

// 进群申请的方式： 1. 邀请， 2. 申请
type GroupJoinSource int

const (
	InviteGroupJoinSource GroupJoinSource = iota + 1
	PutInGroupJoinSource
)

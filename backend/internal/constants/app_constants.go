package constants

// 应用常量。
const (
	AppName                 = "agridispatch"
	APIVersion              = "v1"
	PageSizeDefault         = 10
	PageSizeMax             = 100
	OverviewCacheKey        = "agridispatch:overview"
	OverviewCacheTTLSeconds = 30
	WSPushIntervalSeconds   = 5
)

// 角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 农机状态
const (
	MachineIdle    = "空闲"
	MachineWorking = "作业中"
	MachineRepair  = "维修中"
)

// 任务状态
const (
	TaskPending    = "待派单"
	TaskDispatched = "已派单"
	TaskDone       = "已完成"
)

// 错误码
const (
	CodeOK              = 0
	CodeBadRequest      = 40000
	CodeUnauthorized    = 40100
	CodeForbidden       = 40300
	CodeNotFound        = 40400
	CodeInternalError   = 50000
	CodeTooManyRequests = 42900
)

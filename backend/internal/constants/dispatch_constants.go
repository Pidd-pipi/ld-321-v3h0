package constants

// 驾驶员状态
const (
	DriverOnDuty    = "在岗"
	DriverAvailable = "可派单"
	DriverWorking   = "作业中"
	DriverResting   = "休息"
)

// 派单动作
const (
	DispatchActionConfirm  = "派单"
	DispatchActionReassign = "改派"
	DefaultDispatchReason  = "系统按空闲度和驾驶员排班完成推荐派单"
)

// 派单/改派业务错误码
const (
	CodeDispatchConflict = 40900
)

// 派单/改派错误提示
const (
	ErrMsgTaskNotFound          = "任务不存在"
	ErrMsgMachineNotFound       = "农机不存在"
	ErrMsgDriverNotFound        = "驾驶员不存在"
	ErrMsgTaskNotPending        = "任务当前状态为%s，仅待派单任务可以派单"
	ErrMsgTaskNotDispatched     = "任务当前状态为%s，仅已派单任务可以改派"
	ErrMsgMachineNotIdle        = "农机%s当前状态为%s，仅空闲农机可以派单"
	ErrMsgDriverNotDispatchable = "驾驶员%s当前状态为%s，仅在岗或可派单驾驶员可以派单"
	ErrMsgReassignSameResource  = "新农机或新驾驶员需与当前派单资源不同"
	ErrMsgReassignReasonBlank   = "改派原因不能为空"
)

// 派单/改派成功提示
const (
	MsgDispatchConfirmed = "派单成功，三方状态已同步更新"
	MsgReassigned        = "改派成功，旧资源已释放并原子分配新资源"
)

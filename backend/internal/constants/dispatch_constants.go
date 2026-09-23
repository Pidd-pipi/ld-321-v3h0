package constants

// 驾驶员状态
const (
	DriverIdle     = "在岗"
	DriverDispatch = "可派单"
	DriverWorking  = "作业中"
	DriverResting  = "休息"
)

// 派单动作
const (
	DispatchActionConfirm  = "派单"
	DispatchActionReassign = "改派"
)

// 默认派单原因
const (
	DefaultDispatchReason   = "调度员确认派单"
	DefaultDispatchOperator = "调度员"
	IdleMachineTask         = "可派单"
)

// 派单相关业务错误码
const (
	CodeTaskNotPending     = 40901
	CodeTaskNotDispatched  = 40902
	CodeTaskDone           = 40903
	CodeMachineNotFound    = 40401
	CodeMachineNotIdle     = 40904
	CodeDriverNotFound     = 40402
	CodeDriverUnavailable  = 40905
	CodeReassignSameTarget = 40906
)

// 派单相关业务错误信息（集中维护，禁止在业务代码中写裸字符串）。
const (
	MsgTaskNotPending     = "任务 %s 当前状态为「%s」，仅待派单任务可以派单"
	MsgTaskNotDispatched  = "任务 %s 当前状态为「%s」，仅已派单任务可以改派"
	MsgMachineNotFound    = "农机不存在：%s"
	MsgMachineNotIdle     = "农机 %s 当前状态为「%s」，仅空闲农机可以派单"
	MsgDriverNotFound     = "驾驶员不存在：%s"
	MsgDriverUnavailable  = "驾驶员 %s 当前状态为「%s」，仅在岗或可派单的驾驶员可以派单"
	MsgReassignSameTarget = "新农机与驾驶员与当前派单一致，无需改派"
	MsgReassignReason     = "改派必须填写改派原因"
	MsgOldMachineChanged  = "旧农机 %s 状态已变更，改派终止"
	MsgOldDriverChanged   = "旧驾驶员 %s 状态已变更，改派终止"
	MsgNoRecommended      = "任务未提供推荐资源，请显式指定农机和驾驶员"
)

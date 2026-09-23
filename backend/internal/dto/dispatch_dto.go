package dto

// DispatchConfirmRequest 派单确认请求。
// MachineCode / DriverName 为空时沿用任务推荐资源。
type DispatchConfirmRequest struct {
	MachineCode string `json:"machineCode" validate:"omitempty,max=32"`
	DriverName  string `json:"driverName" validate:"omitempty,max=64"`
	Reason      string `json:"reason" validate:"omitempty,max=255"`
}

// ReassignRequest 改派请求。
// MachineCode / DriverName 为空时沿用当前派单资源；Reason 必填。
type ReassignRequest struct {
	MachineCode string `json:"machineCode" validate:"omitempty,max=32"`
	DriverName  string `json:"driverName" validate:"omitempty,max=64"`
	Reason      string `json:"reason" validate:"required,max=255"`
}

// DispatchResult 派单/改派结果。
type DispatchResult struct {
	TaskID      string `json:"taskId"`
	Status      string `json:"status"`
	MachineCode string `json:"machineCode"`
	DriverName  string `json:"driverName"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Message     string `json:"message"`
}

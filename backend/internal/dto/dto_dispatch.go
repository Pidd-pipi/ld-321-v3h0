package dto

// DispatchConfirmRequest 派单确认请求。
// MachineCode / DriverName 可选；未传时沿用任务的推荐资源。
type DispatchConfirmRequest struct {
	MachineCode string `json:"machineCode"`
	DriverName  string `json:"driverName"`
	Reason      string `json:"reason"`
}

// DispatchReassignRequest 已派单任务改派请求。Reason 必填。
// MachineCode / DriverName 可选；未传的沿用任务的推荐资源。
type DispatchReassignRequest struct {
	MachineCode string `json:"machineCode"`
	DriverName  string `json:"driverName"`
	Reason      string `json:"reason"`
}

// DispatchResponse 派单/改派结果。
type DispatchResponse struct {
	TaskID      string `json:"taskId"`
	Status      string `json:"status"`
	MachineCode string `json:"machineCode"`
	DriverName  string `json:"driverName"`
	Reason      string `json:"reason"`
	Action      string `json:"action"`
	Message     string `json:"message"`
}

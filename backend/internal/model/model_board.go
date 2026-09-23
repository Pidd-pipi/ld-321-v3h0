package model

// DispatchBoard 调度看板。
type DispatchBoard struct {
	TodayTodos      int      `json:"todayTodos"`
	IdleMachines    int      `json:"idleMachines"`
	WorkingMachines []string `json:"workingMachines"`
	DueMaintenance  []string `json:"dueMaintenance"`
	SevenDayAreas   []int    `json:"sevenDayAreas"`
	TrendLabels     []string `json:"trendLabels"`
}

// Stats 作业统计。
type Stats struct {
	TotalAreaMu float64 `json:"totalAreaMu"`
	TotalHours  float64 `json:"totalHours"`
	FuelCost    float64 `json:"fuelCost"`
}

// FarmOverview 调度看板总览。
type FarmOverview struct {
	Items       []DashboardItem       `json:"items"`
	Machines    []Machine             `json:"machines"`
	Tasks       []FarmTask            `json:"tasks"`
	Tracks      []TrackPoint          `json:"tracks"`
	Records     []WorkRecord          `json:"records"`
	Maintenance []MaintenanceReminder `json:"maintenance"`
	Drivers     []Driver              `json:"drivers"`
	Board       DispatchBoard         `json:"board"`
	Stats       Stats                 `json:"stats"`
}

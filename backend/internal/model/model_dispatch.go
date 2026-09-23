package model

import "time"

// DispatchRecord 派单/改派记录。
type DispatchRecord struct {
	ID              string    `gorm:"primaryKey;size:32" json:"id"`
	TaskID          string    `gorm:"size:32;index" json:"taskId"`
	Action          string    `gorm:"size:16" json:"action"`
	MachineCode     string    `gorm:"size:32" json:"machineCode"`
	DriverName      string    `gorm:"size:64" json:"driverName"`
	PreviousMachine string    `gorm:"size:32" json:"previousMachine"`
	PreviousDriver  string    `gorm:"size:64" json:"previousDriver"`
	Reason          string    `gorm:"size:255" json:"reason"`
	Operator        string    `gorm:"size:64" json:"operator"`
	CreatedAt       time.Time `json:"createdAt"`
}

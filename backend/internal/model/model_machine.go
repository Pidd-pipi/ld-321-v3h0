package model

import "time"

// Machine 农机档案。
type Machine struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	Code        string    `gorm:"size:32;uniqueIndex" json:"code"`
	Name        string    `gorm:"size:64" json:"name"`
	Model       string    `gorm:"size:64" json:"model"`
	PurchasedAt string    `gorm:"size:32" json:"purchasedAt"`
	Horsepower  int       `json:"horsepower"`
	Field       string    `gorm:"size:64" json:"field"`
	Status      string    `gorm:"size:20;index" json:"status"`
	QRCode      string    `gorm:"size:64" json:"qrCode"`
	PhotoURL    string    `gorm:"size:255" json:"photoUrl"`
	WorkHours   float64   `json:"workHours"`
	CurrentTask string    `gorm:"size:64" json:"currentTask"`
	CreatedAt   time.Time `json:"createdAt"`
}

// FarmTask 作业任务。
type FarmTask struct {
	ID                 string          `gorm:"primaryKey;size:32" json:"id"`
	Type               string          `gorm:"size:32" json:"type"`
	Field              string          `gorm:"size:64" json:"field"`
	AreaMu             float64         `json:"areaMu"`
	EstimatedHours     float64         `json:"estimatedHours"`
	Status             string          `gorm:"size:20;index" json:"status"`
	Priority           string          `gorm:"size:16" json:"priority"`
	RecommendedMachine string          `gorm:"size:64" json:"recommendedMachine"`
	RecommendedDriver  string          `gorm:"size:64" json:"recommendedDriver"`
	AssignedMachine    string          `gorm:"size:64" json:"assignedMachine"`
	AssignedDriver     string          `gorm:"size:64" json:"assignedDriver"`
	PlannedWindow      string          `gorm:"size:64" json:"plannedWindow"`
	CreatedAt          time.Time       `json:"createdAt"`
	LatestRecord       *DispatchRecord `gorm:"-" json:"latestRecord,omitempty"`
}

// DispatchRecord 派单/改派记录。
type DispatchRecord struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	TaskID      string    `gorm:"size:32;index" json:"taskId"`
	Action      string    `gorm:"size:16" json:"action"`
	MachineCode string    `gorm:"size:32" json:"machineCode"`
	DriverName  string    `gorm:"size:64" json:"driverName"`
	Reason      string    `gorm:"size:255" json:"reason"`
	PrevMachine string    `gorm:"size:32" json:"prevMachine"`
	PrevDriver  string    `gorm:"size:64" json:"prevDriver"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TrackPoint 农机实时轨迹点。
type TrackPoint struct {
	ID            uint      `gorm:"primaryKey" json:"-"`
	MachineCode   string    `gorm:"size:32;index" json:"machineCode"`
	TaskType      string    `gorm:"size:32" json:"taskType"`
	CapturedAt    string    `gorm:"size:32" json:"capturedAt"`
	Longitude     float64   `json:"longitude"`
	Latitude      float64   `json:"latitude"`
	Speed         float64   `json:"speed"`
	FieldBoundary string    `gorm:"size:64" json:"fieldBoundary"`
	CreatedAt     time.Time `json:"-"`
}

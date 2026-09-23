package model

import "time"

// WorkRecord 作业记录。
type WorkRecord struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	MachineCode string    `gorm:"size:32;index" json:"machineCode"`
	DriverName  string    `gorm:"size:64" json:"driverName"`
	WorkDate    string    `gorm:"size:32" json:"workDate"`
	TaskType    string    `gorm:"size:32" json:"taskType"`
	ActualHours float64   `json:"actualHours"`
	FuelLiters  float64   `json:"fuelLiters"`
	AreaMu      float64   `json:"areaMu"`
	FuelCost    float64   `json:"fuelCost"`
	CreatedAt   time.Time `json:"-"`
}

// MaintenanceReminder 保养提醒。
type MaintenanceReminder struct {
	ID                string    `gorm:"primaryKey;size:32" json:"id"`
	MachineCode       string    `gorm:"size:32;index" json:"machineCode"`
	Title             string    `gorm:"size:128" json:"title"`
	DueDate           string    `gorm:"size:32" json:"dueDate"`
	RemainingHours    float64   `json:"remainingHours"`
	Level             string    `gorm:"size:16" json:"level"`
	LastServiceRecord string    `gorm:"size:128" json:"lastServiceRecord"`
	CreatedAt         time.Time `json:"-"`
}

// Driver 驾驶员。
type Driver struct {
	ID           string    `gorm:"primaryKey;size:32" json:"id"`
	Name         string    `gorm:"size:64" json:"name"`
	LicenseNo    string    `gorm:"size:32" json:"licenseNo"`
	Phone        string    `gorm:"size:32" json:"phone"`
	Shift        string    `gorm:"size:16" json:"shift"`
	RestDay      string    `gorm:"size:16" json:"restDay"`
	MonthAreaMu  float64   `json:"monthAreaMu"`
	Rating       float64   `json:"rating"`
	Status       string    `gorm:"size:16" json:"status"`
	CreatedAt    time.Time `json:"-"`
}

// DashboardItem 功能模块卡片。
type DashboardItem struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	Title       string    `gorm:"size:64" json:"title"`
	Description string    `gorm:"size:128" json:"description"`
	Status      string    `gorm:"size:16" json:"status"`
	Score       int       `json:"score"`
	CreatedAt   time.Time `json:"-"`
}

// User 平台用户。
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:20;not null" json:"role"`
	RealName     string    `gorm:"size:64" json:"realName"`
	CreatedAt    time.Time `json:"createdAt"`
}

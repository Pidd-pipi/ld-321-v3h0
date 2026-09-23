package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/agridispatch/agridispatch/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect 连接 MySQL（带启动期重试）并自动迁移。
func Connect(dsn string, maxOpen, maxIdle, connMaxLifetime, retryCount, retryInterval int) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for i := 0; i <= retryCount; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
		if err == nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil && sqlDB.Ping() == nil {
				break
			}
		}
		if i == retryCount {
			return nil, fmt.Errorf("open mysql after %d retries: %w", retryCount, err)
		}
		slog.Warn("database not ready, retrying", "attempt", i+1, "err", err)
		time.Sleep(time.Duration(retryInterval) * time.Second)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
	if err := db.AutoMigrate(
		&model.User{},
		&model.Machine{},
		&model.FarmTask{},
		&model.TrackPoint{},
		&model.WorkRecord{},
		&model.MaintenanceReminder{},
		&model.Driver{},
		&model.DashboardItem{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}

// Seed 幂等种子数据。
func Seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	// 管理员
	admin := &model.User{Username: "admin", PasswordHash: hashPwd("admin123"), Role: "admin", RealName: "系统管理员"}
	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	// 功能模块卡片
	items := []model.DashboardItem{
		{ID: "module-1", Title: "农机档案", Description: "唯一编号、二维码、照片和状态筛选", Status: "运行中", Score: 92},
		{ID: "module-2", Title: "任务调度", Description: "推荐空闲农机和驾驶员并支持派单", Status: "运行中", Score: 90},
		{ID: "module-3", Title: "轨迹地图", Description: "实时位置、历史轨迹和地块边界", Status: "运行中", Score: 88},
		{ID: "module-4", Title: "作业统计", Description: "日报、月报、油耗成本和面积统计", Status: "运行中", Score: 91},
		{ID: "module-5", Title: "保养提醒", Description: "按工时和日历周期生成预警", Status: "运行中", Score: 87},
		{ID: "module-6", Title: "驾驶员", Description: "证照、排班、评价和历史作业量", Status: "运行中", Score: 89},
		{ID: "module-7", Title: "调度看板", Description: "待办、空闲、作业中和七日趋势", Status: "运行中", Score: 93},
	}
	if err := db.Create(&items).Error; err != nil {
		return fmt.Errorf("seed items: %w", err)
	}
	// 农机
	machines := []model.Machine{
		{ID: "m1", Code: "NJ-2026-001", Name: "东方红 1804", Model: "LX1804", PurchasedAt: "2023-03-12", Horsepower: 180, Field: "北岭 1 号田", Status: "作业中", QRCode: "QR-NJ-001", PhotoURL: "/assets/machine-tractor.jpg", WorkHours: 284.5, CurrentTask: "春耕翻地"},
		{ID: "m2", Code: "NJ-2026-002", Name: "雷沃谷神收割机", Model: "GE80S", PurchasedAt: "2022-09-18", Horsepower: 160, Field: "南湾稻田", Status: "空闲", QRCode: "QR-NJ-002", PhotoURL: "/assets/machine-harvester.jpg", WorkHours: 412.0, CurrentTask: "可派单"},
		{ID: "m3", Code: "NJ-2026-003", Name: "中联履带拖拉机", Model: "RK140", PurchasedAt: "2024-01-06", Horsepower: 140, Field: "西坡旱地", Status: "维修中", QRCode: "QR-NJ-003", PhotoURL: "/assets/machine-crawler.jpg", WorkHours: 98.0, CurrentTask: "液压检修"},
	}
	if err := db.Create(&machines).Error; err != nil {
		return fmt.Errorf("seed machines: %w", err)
	}
	// 任务
	tasks := []model.FarmTask{
		{ID: "t1", Type: "耕地", Field: "北岭 1 号田", AreaMu: 180, EstimatedHours: 9.5, Status: "已派单", Priority: "高", RecommendedMachine: "NJ-2026-001", RecommendedDriver: "周明", PlannedWindow: "今日 08:00-18:00"},
		{ID: "t2", Type: "播种", Field: "西坡旱地", AreaMu: 96, EstimatedHours: 6.0, Status: "待派单", Priority: "中", RecommendedMachine: "NJ-2026-002", RecommendedDriver: "何燕", PlannedWindow: "明日 07:30-14:00"},
		{ID: "t3", Type: "施肥", Field: "南湾稻田", AreaMu: 132, EstimatedHours: 5.5, Status: "待派单", Priority: "中", RecommendedMachine: "NJ-2026-002", RecommendedDriver: "刘强", PlannedWindow: "今日 14:00-20:00"},
		{ID: "t4", Type: "收割", Field: "东河麦田", AreaMu: 210, EstimatedHours: 11.0, Status: "已完成", Priority: "高", RecommendedMachine: "NJ-2026-004", RecommendedDriver: "周明", PlannedWindow: "昨日 06:30-17:30"},
	}
	if err := db.Create(&tasks).Error; err != nil {
		return fmt.Errorf("seed tasks: %w", err)
	}
	// 轨迹
	tracks := []model.TrackPoint{
		{MachineCode: "NJ-2026-001", TaskType: "耕地", CapturedAt: "2026-05-31 08:10", Longitude: 116.316, Latitude: 39.985, Speed: 8.2, FieldBoundary: "北岭 1 号田"},
		{MachineCode: "NJ-2026-001", TaskType: "耕地", CapturedAt: "2026-05-31 09:20", Longitude: 116.322, Latitude: 39.988, Speed: 7.6, FieldBoundary: "北岭 1 号田"},
		{MachineCode: "NJ-2026-001", TaskType: "耕地", CapturedAt: "2026-05-31 10:30", Longitude: 116.329, Latitude: 39.991, Speed: 8.8, FieldBoundary: "北岭 1 号田"},
		{MachineCode: "NJ-2026-002", TaskType: "播种", CapturedAt: "2026-05-30 15:00", Longitude: 116.301, Latitude: 39.972, Speed: 6.4, FieldBoundary: "西坡旱地"},
	}
	if err := db.Create(&tracks).Error; err != nil {
		return fmt.Errorf("seed tracks: %w", err)
	}
	// 作业记录
	records := []model.WorkRecord{
		{ID: "r1", MachineCode: "NJ-2026-001", DriverName: "周明", WorkDate: "2026-05-31", TaskType: "耕地", ActualHours: 8.5, FuelLiters: 76, AreaMu: 156, FuelCost: 562.4},
		{ID: "r2", MachineCode: "NJ-2026-002", DriverName: "何燕", WorkDate: "2026-05-30", TaskType: "播种", ActualHours: 5.8, FuelLiters: 43, AreaMu: 88, FuelCost: 318.2},
		{ID: "r3", MachineCode: "NJ-2026-004", DriverName: "刘强", WorkDate: "2026-05-29", TaskType: "收割", ActualHours: 10.2, FuelLiters: 91, AreaMu: 203, FuelCost: 673.4},
		{ID: "r4", MachineCode: "NJ-2026-001", DriverName: "周明", WorkDate: "2026-05-28", TaskType: "施肥", ActualHours: 6.4, FuelLiters: 55, AreaMu: 122, FuelCost: 407.0},
	}
	if err := db.Create(&records).Error; err != nil {
		return fmt.Errorf("seed records: %w", err)
	}
	// 保养提醒
	reminders := []model.MaintenanceReminder{
		{ID: "s1", MachineCode: "NJ-2026-001", Title: "100 小时换机油", DueDate: "2026-06-05", RemainingHours: 16, Level: "warning", LastServiceRecord: "2026-04-28 已更换滤芯"},
		{ID: "s2", MachineCode: "NJ-2026-003", Title: "液压系统复检", DueDate: "2026-06-02", RemainingHours: 0, Level: "danger", LastServiceRecord: "2026-05-25 漏油维修"},
		{ID: "s3", MachineCode: "NJ-2026-002", Title: "刀盘检查", DueDate: "2026-06-12", RemainingHours: 42, Level: "normal", LastServiceRecord: "2026-05-12 例行保养"},
	}
	if err := db.Create(&reminders).Error; err != nil {
		return fmt.Errorf("seed reminders: %w", err)
	}
	// 驾驶员
	drivers := []model.Driver{
		{ID: "d1", Name: "周明", LicenseNo: "A2-4101811990", Phone: "13800010001", Shift: "早班", RestDay: "周日", MonthAreaMu: 486, Rating: 4.8, Status: "在岗"},
		{ID: "d2", Name: "何燕", LicenseNo: "B2-4101811992", Phone: "13800010002", Shift: "中班", RestDay: "周三", MonthAreaMu: 318, Rating: 4.7, Status: "可派单"},
		{ID: "d3", Name: "刘强", LicenseNo: "A1-4101811988", Phone: "13800010003", Shift: "夜班", RestDay: "周五", MonthAreaMu: 402, Rating: 4.6, Status: "休息"},
	}
	if err := db.Create(&drivers).Error; err != nil {
		return fmt.Errorf("seed drivers: %w", err)
	}
	slog.Info("seeded agridispatch demo data")
	return nil
}

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	apperrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// newDispatchTestDB 构造内存 SQLite 并写入派单测试夹具。
func newDispatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:dispatch-%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Machine{}, &model.FarmTask{}, &model.Driver{}, &model.DispatchRecord{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	machines := []model.Machine{
		{ID: "m1", Code: "NJ-2026-001", Name: "东方红 1804", Status: constants.MachineWorking, CurrentTask: "耕地 北岭 1 号田"},
		{ID: "m2", Code: "NJ-2026-002", Name: "雷沃谷神收割机", Status: constants.MachineIdle, CurrentTask: constants.IdleMachineTask},
		{ID: "m3", Code: "NJ-2026-003", Name: "中联履带拖拉机", Status: constants.MachineRepair, CurrentTask: "液压检修"},
		{ID: "m5", Code: "NJ-2026-005", Name: "沃得奥龙拖拉机", Status: constants.MachineIdle, CurrentTask: constants.IdleMachineTask},
	}
	if err := db.Create(&machines).Error; err != nil {
		t.Fatalf("seed machines: %v", err)
	}
	drivers := []model.Driver{
		{ID: "d1", Name: "周明", Status: constants.DriverWorking},
		{ID: "d2", Name: "何燕", Status: constants.DriverDispatch},
		{ID: "d3", Name: "刘强", Status: constants.DriverResting},
		{ID: "d4", Name: "王芳", Status: constants.DriverIdle},
	}
	if err := db.Create(&drivers).Error; err != nil {
		t.Fatalf("seed drivers: %v", err)
	}
	tasks := []model.FarmTask{
		{ID: "t-pending", Type: "播种", Field: "西坡旱地", Status: constants.TaskPending, RecommendedMachine: "NJ-2026-002", RecommendedDriver: "何燕"},
		{ID: "t-pending-rest", Type: "施肥", Field: "南湾稻田", Status: constants.TaskPending, RecommendedMachine: "NJ-2026-002", RecommendedDriver: "刘强"},
		{ID: "t-dispatched", Type: "耕地", Field: "北岭 1 号田", Status: constants.TaskDispatched, RecommendedMachine: "NJ-2026-002", RecommendedDriver: "王芳", AssignedMachine: "NJ-2026-001", AssignedDriver: "周明", DispatchReason: "首次派单"},
		{ID: "t-done", Type: "收割", Field: "东河麦田", Status: constants.TaskDone},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("seed tasks: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM dispatch_records").Error
	})
	return db
}

func newDispatchService(t *testing.T) (*DispatchService, *gorm.DB) {
	db := newDispatchTestDB(t)
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = rdb.Close() })
	logger := slog.Default()
	dashSvc := NewDashboardService(repository.NewDashboardRepository(db), rdb, logger)
	return NewDispatchService(repository.NewDispatchRepository(db), dashSvc, logger), db
}

func bizCode(t *testing.T, err error) int {
	t.Helper()
	var bizErr *apperrors.BusinessError
	if errors.As(err, &bizErr) {
		return bizErr.Code
	}
	t.Fatalf("expected BusinessError, got %T: %v", err, err)
	return 0
}

func reloadState(t *testing.T, db *gorm.DB, taskID string) (model.FarmTask, model.Machine, model.Machine, model.Driver, model.Driver) {
	t.Helper()
	var task model.FarmTask
	if err := db.First(&task, "id = ?", taskID).Error; t.Failed() {
		t.Fatalf("reload task: %v", err)
	}
	var m001, m005 model.Machine
	db.First(&m001, "code = ?", "NJ-2026-001")
	db.First(&m005, "code = ?", "NJ-2026-005")
	var dZhou, dWang model.Driver
	db.First(&dZhou, "name = ?", "周明")
	db.First(&dWang, "name = ?", "王芳")
	return task, m001, m005, dZhou, dWang
}

func recordCount(db *gorm.DB) int64 {
	var n int64
	db.Model(&model.DispatchRecord{}).Count(&n)
	return n
}

func TestDispatchConfirm(t *testing.T) {
	ctx := context.Background()

	t.Run("默认沿用推荐资源并保存原因", func(t *testing.T) {
		svc, db := newDispatchService(t)
		rec, err := svc.Confirm(ctx, "t-pending", "", "", "农时紧张立即作业")
		if err != nil {
			t.Fatalf("Confirm: %v", err)
		}
		if rec.Action != constants.DispatchActionConfirm || rec.MachineCode != "NJ-2026-002" || rec.DriverName != "何燕" || rec.Reason != "农时紧张立即作业" {
			t.Fatalf("record mismatch: %+v", rec)
		}
		var task model.FarmTask
		db.First(&task, "id = ?", "t-pending")
		if task.Status != constants.TaskDispatched || task.AssignedMachine != "NJ-2026-002" || task.AssignedDriver != "何燕" || task.DispatchReason != "农时紧张立即作业" {
			t.Fatalf("task mismatch: %+v", task)
		}
		var machine model.Machine
		db.First(&machine, "code = ?", "NJ-2026-002")
		if machine.Status != constants.MachineWorking {
			t.Fatalf("machine status = %s", machine.Status)
		}
		var driver model.Driver
		db.First(&driver, "name = ?", "何燕")
		if driver.Status != constants.DriverWorking {
			t.Fatalf("driver status = %s", driver.Status)
		}
	})

	t.Run("显式指定农机和驾驶员", func(t *testing.T) {
		svc, db := newDispatchService(t)
		rec, err := svc.Confirm(ctx, "t-pending", "NJ-2026-005", "王芳", "")
		if err != nil {
			t.Fatalf("Confirm: %v", err)
		}
		if rec.MachineCode != "NJ-2026-005" || rec.DriverName != "王芳" {
			t.Fatalf("record mismatch: %+v", rec)
		}
		if rec.Reason != constants.DefaultDispatchReason {
			t.Fatalf("default reason = %s", rec.Reason)
		}
		var m2 model.Machine
		db.First(&m2, "code = ?", "NJ-2026-002")
		if m2.Status != constants.MachineIdle {
			t.Fatalf("recommended machine should stay idle, got %s", m2.Status)
		}
	})

	failureCases := []struct {
		name        string
		taskID      string
		machine     string
		driver      string
		wantCode    int
		description string
	}{
		{"任务已派单不可重复派单", "t-dispatched", "", "", constants.CodeTaskNotPending, "40901"},
		{"任务已完成不可派单", "t-done", "", "", constants.CodeTaskNotPending, "40901"},
		{"农机作业中不可派单", "t-pending", "NJ-2026-001", "王芳", constants.CodeMachineNotIdle, "40904"},
		{"农机维修中不可派单", "t-pending", "NJ-2026-003", "王芳", constants.CodeMachineNotIdle, "40904"},
		{"农机不存在", "t-pending", "NJ-NOPE", "王芳", constants.CodeMachineNotFound, "40401"},
		{"驾驶员休息不可派单", "t-pending", "NJ-2026-005", "刘强", constants.CodeDriverUnavailable, "40905"},
		{"驾驶员不存在", "t-pending", "NJ-2026-005", "不存在的人", constants.CodeDriverNotFound, "40402"},
		{"推荐驾驶员休息同样拒绝", "t-pending-rest", "", "", constants.CodeDriverUnavailable, "40905"},
	}
	for _, tc := range failureCases {
		t.Run("失败不改状态_"+tc.name, func(t *testing.T) {
			svc, db := newDispatchService(t)
			before := recordCount(db)
			_, err := svc.Confirm(ctx, tc.taskID, tc.machine, tc.driver, "原因")
			if err == nil {
				t.Fatalf("expected error code %s", tc.description)
			}
			if code := bizCode(t, err); code != tc.wantCode {
				t.Fatalf("error code = %d, want %d (%v)", code, tc.wantCode, err)
			}
			if recordCount(db) != before {
				t.Fatal("dispatch record must not be saved on failure")
			}
			taskID := tc.taskID
			if taskID == "t-pending" || taskID == "t-pending-rest" {
				var task model.FarmTask
				db.First(&task, "id = ?", taskID)
				if task.Status != constants.TaskPending {
					t.Fatalf("task status changed to %s", task.Status)
				}
			}
			for _, code := range []string{"NJ-2026-001", "NJ-2026-002", "NJ-2026-003", "NJ-2026-005"} {
				var m model.Machine
				db.First(&m, "code = ?", code)
				want := constants.MachineIdle
				if code == "NJ-2026-001" {
					want = constants.MachineWorking
				}
				if code == "NJ-2026-003" {
					want = constants.MachineRepair
				}
				if m.Status != want {
					t.Fatalf("machine %s status = %s, want %s", code, m.Status, want)
				}
			}
			for _, name := range []string{"周明", "何燕", "王芳"} {
				var d model.Driver
				db.First(&d, "name = ?", name)
				want := constants.DriverWorking
				if name == "何燕" {
					want = constants.DriverDispatch
				}
				if name == "王芳" {
					want = constants.DriverIdle
				}
				if d.Status != want {
					t.Fatalf("driver %s status = %s, want %s", name, d.Status, want)
				}
			}
		})
	}
}

func TestDispatchReassign(t *testing.T) {
	ctx := context.Background()

	t.Run("任务不存在返回哨兵错误", func(t *testing.T) {
		svc, _ := newDispatchService(t)
		_, err := svc.Confirm(ctx, "t-missing", "", "", "原因")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		_, err = svc.Reassign(ctx, "t-missing", "", "", "原因")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("原子释放旧资源并分配新资源", func(t *testing.T) {
		svc, db := newDispatchService(t)
		before := recordCount(db)
		rec, err := svc.Reassign(ctx, "t-dispatched", "NJ-2026-005", "王芳", "原农机转抢修北岭地块")
		if err != nil {
			t.Fatalf("Reassign: %v", err)
		}
		if rec.Action != constants.DispatchActionReassign || rec.PreviousMachine != "NJ-2026-001" || rec.PreviousDriver != "周明" {
			t.Fatalf("record mismatch: %+v", rec)
		}
		if recordCount(db) != before+1 {
			t.Fatal("exactly one reassign record expected")
		}
		task, m001, m005, dZhou, dWang := reloadState(t, db, "t-dispatched")
		if task.AssignedMachine != "NJ-2026-005" || task.AssignedDriver != "王芳" || task.DispatchReason != "原农机转抢修北岭地块" || task.Status != constants.TaskDispatched {
			t.Fatalf("task mismatch: %+v", task)
		}
		if m001.Status != constants.MachineIdle || m001.CurrentTask != constants.IdleMachineTask {
			t.Fatalf("old machine not released: %+v", m001)
		}
		if m005.Status != constants.MachineWorking {
			t.Fatalf("new machine not occupied: %+v", m005)
		}
		if dZhou.Status != constants.DriverDispatch {
			t.Fatalf("old driver status = %s", dZhou.Status)
		}
		if dWang.Status != constants.DriverWorking {
			t.Fatalf("new driver status = %s", dWang.Status)
		}
	})

	t.Run("仅换驾驶员时农机不动", func(t *testing.T) {
		svc, db := newDispatchService(t)
		_, err := svc.Reassign(ctx, "t-dispatched", "NJ-2026-001", "王芳", "周明临时请假")
		if err != nil {
			t.Fatalf("Reassign: %v", err)
		}
		task, m001, _, dZhou, dWang := reloadState(t, db, "t-dispatched")
		if task.AssignedMachine != "NJ-2026-001" || task.AssignedDriver != "王芳" {
			t.Fatalf("task mismatch: %+v", task)
		}
		if m001.Status != constants.MachineWorking {
			t.Fatalf("machine should stay working, got %s", m001.Status)
		}
		if dZhou.Status != constants.DriverDispatch || dWang.Status != constants.DriverWorking {
			t.Fatalf("driver swap wrong: zhou=%s wang=%s", dZhou.Status, dWang.Status)
		}
	})

	t.Run("新资源不可用时整单回滚", func(t *testing.T) {
		svc, db := newDispatchService(t)
		before := recordCount(db)
		_, err := svc.Reassign(ctx, "t-dispatched", "NJ-2026-003", "王芳", "想改派到维修农机")
		if err == nil || bizCode(t, err) != constants.CodeMachineNotIdle {
			t.Fatalf("expected CodeMachineNotIdle, got %v", err)
		}
		if recordCount(db) != before {
			t.Fatal("no record expected on failed reassign")
		}
		task, m001, m005, dZhou, dWang := reloadState(t, db, "t-dispatched")
		if task.AssignedMachine != "NJ-2026-001" || task.AssignedDriver != "周明" || task.DispatchReason != "首次派单" {
			t.Fatalf("task changed on failed reassign: %+v", task)
		}
		if m001.Status != constants.MachineWorking || m005.Status != constants.MachineIdle {
			t.Fatalf("machine states changed: old=%s new=%s", m001.Status, m005.Status)
		}
		if dZhou.Status != constants.DriverWorking || dWang.Status != constants.DriverIdle {
			t.Fatalf("driver states changed: zhou=%s wang=%s", dZhou.Status, dWang.Status)
		}
	})

	t.Run("新驾驶员休息时整单回滚", func(t *testing.T) {
		svc, db := newDispatchService(t)
		_, err := svc.Reassign(ctx, "t-dispatched", "NJ-2026-005", "刘强", "改派给休息驾驶员应失败")
		if err == nil || bizCode(t, err) != constants.CodeDriverUnavailable {
			t.Fatalf("expected CodeDriverUnavailable, got %v", err)
		}
		var m005 model.Machine
		db.First(&m005, "code = ?", "NJ-2026-005")
		if m005.Status != constants.MachineIdle {
			t.Fatalf("new machine occupied despite rollback: %s", m005.Status)
		}
	})

	invalidCases := []struct {
		name     string
		taskID   string
		machine  string
		driver   string
		reason   string
		wantCode int
	}{
		{"改派原因必填", "t-dispatched", "NJ-2026-005", "王芳", "  ", constants.CodeBadRequest},
		{"待派单任务不能改派", "t-pending", "NJ-2026-005", "王芳", "原因", constants.CodeTaskNotDispatched},
		{"已完成任务不能改派", "t-done", "NJ-2026-005", "王芳", "原因", constants.CodeTaskNotDispatched},
		{"新旧资源一致无需改派", "t-dispatched", "NJ-2026-001", "周明", "原因", constants.CodeReassignSameTarget},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := newDispatchService(t)
			before := recordCount(db)
			_, err := svc.Reassign(ctx, tc.taskID, tc.machine, tc.driver, tc.reason)
			if err == nil {
				t.Fatal("expected error")
			}
			if code := bizCode(t, err); code != tc.wantCode {
				t.Fatalf("error code = %d, want %d", code, tc.wantCode)
			}
			if recordCount(db) != before {
				t.Fatal("no record expected on rejected reassign")
			}
		})
	}
}

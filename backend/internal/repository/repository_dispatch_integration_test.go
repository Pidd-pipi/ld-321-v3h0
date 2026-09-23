//go:build integration

package repository

import (
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newSQLiteRepo(t *testing.T) (*DispatchRepository, *gorm.DB) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.FarmTask{}, &model.Machine{}, &model.Driver{}, &model.DispatchRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewDispatchRepository(db), db
}

func seedDispatchFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	machines := []model.Machine{
		{ID: "m1", Code: "NJ-1", Name: "M1", Status: constants.MachineIdle},
		{ID: "m2", Code: "NJ-2", Name: "M2", Status: constants.MachineWorking, CurrentTask: "别的任务"},
	}
	drivers := []model.Driver{
		{ID: "d1", Name: "在岗甲", Status: constants.DriverOnDuty},
		{ID: "d2", Name: "休息乙", Status: constants.DriverResting},
	}
	tasks := []model.FarmTask{
		{ID: "t-pending", Type: "耕地", Field: "北田", Status: constants.TaskPending, RecommendedMachine: "NJ-1", RecommendedDriver: "在岗甲"},
		{ID: "t-dispatched", Type: "播种", Field: "南田", Status: constants.TaskDispatched, AssignedMachine: "NJ-2", AssignedDriver: "在岗甲", RecommendedMachine: "NJ-2", RecommendedDriver: "在岗甲"},
		{ID: "t-done", Type: "收割", Field: "东田", Status: constants.TaskDone},
	}
	if err := db.Create(&machines).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&drivers).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_ConfirmDispatch_Success(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)

	if _, _, err := repo.ConfirmDispatch("t-pending", DispatchParams{}); err != nil {
		t.Fatalf("ConfirmDispatch() err = %v", err)
	}
	var task model.FarmTask
	db.First(&task, "id = ?", "t-pending")
	if task.Status != constants.TaskDispatched || task.AssignedMachine != "NJ-1" || task.AssignedDriver != "在岗甲" {
		t.Errorf("task = %+v", task)
	}
	var m model.Machine
	db.First(&m, "code = ?", "NJ-1")
	if m.Status != constants.MachineWorking {
		t.Errorf("machine status = %s", m.Status)
	}
	var d model.Driver
	db.First(&d, "name = ?", "在岗甲")
	if d.Status != constants.DriverWorking {
		t.Errorf("driver status = %s", d.Status)
	}
	var recs []model.DispatchRecord
	if err := db.Where("task_id = ?", "t-pending").Find(&recs).Error; err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Action != constants.DispatchActionConfirm {
		t.Errorf("records = %+v", recs)
	}
}

func TestIntegration_ConfirmDispatch_BusyMachine_NoStateChange(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)

	// 指定正在作业的农机 NJ-2，必须失败。
	_, _, err := repo.ConfirmDispatch("t-pending", DispatchParams{MachineCode: "NJ-2"})
	if err == nil {
		t.Fatal("期望农机非空闲错误")
	}
	assertNoStateChange(t, db)
}

func TestIntegration_ConfirmDispatch_RestingDriver_NoStateChange(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)

	_, _, err := repo.ConfirmDispatch("t-pending", DispatchParams{DriverName: "休息乙"})
	if err == nil {
		t.Fatal("期望驾驶员不可派单错误")
	}
	assertNoStateChange(t, db)
}

func TestIntegration_ConfirmDispatch_DoneTask_NoStateChange(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)

	if _, _, err := repo.ConfirmDispatch("t-done", DispatchParams{}); err == nil {
		t.Fatal("期望任务状态冲突错误")
	}
	assertNoStateChange(t, db)
}

func TestIntegration_Reassign_Success(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)
	// 先把待派单任务派掉，腾出一台空闲农机用于改派校验不合适；直接准备：NJ-3 空闲 + 可派单驾驶员。
	if err := db.Create(&[]model.Machine{{ID: "m3", Code: "NJ-3", Status: constants.MachineIdle}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.Driver{{ID: "d3", Name: "可派丙", Status: constants.DriverAvailable}}).Error; err != nil {
		t.Fatal(err)
	}

	if _, _, err := repo.Reassign("t-dispatched", ReassignParams{MachineCode: "NJ-3", DriverName: "可派丙", Reason: "原农机故障"}); err != nil {
		t.Fatalf("Reassign() err = %v", err)
	}
	var task model.FarmTask
	db.First(&task, "id = ?", "t-dispatched")
	if task.AssignedMachine != "NJ-3" || task.AssignedDriver != "可派丙" {
		t.Errorf("task = %+v", task)
	}
	var oldM model.Machine
	db.First(&oldM, "code = ?", "NJ-2")
	if oldM.Status != constants.MachineIdle || oldM.CurrentTask != "" {
		t.Errorf("旧农机未释放: %+v", oldM)
	}
	var newM model.Machine
	db.First(&newM, "code = ?", "NJ-3")
	if newM.Status != constants.MachineWorking {
		t.Errorf("新农机未占用: %+v", newM)
	}
	var oldD, newD model.Driver
	db.First(&oldD, "name = ?", "在岗甲")
	db.First(&newD, "name = ?", "可派丙")
	if oldD.Status != constants.DriverAvailable {
		t.Errorf("旧驾驶员状态 = %s", oldD.Status)
	}
	if newD.Status != constants.DriverWorking {
		t.Errorf("新驾驶员状态 = %s", newD.Status)
	}
	var recs []model.DispatchRecord
	db.Where("task_id = ?", "t-dispatched").Order("created_at DESC").Find(&recs)
	if len(recs) != 1 || recs[0].Action != constants.DispatchActionReassign ||
		recs[0].PrevMachine != "NJ-2" || recs[0].PrevDriver != "在岗甲" || recs[0].Reason != "原农机故障" {
		t.Errorf("改派记录 = %+v", recs)
	}
}

func TestIntegration_Reassign_BusyTarget_NoStateChange(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)

	// 已派给 NJ-2/在岗甲；尝试改派给正在作业的 NJ-2（相同资源）再尝试新驾驶员休息乙。
	if _, _, err := repo.Reassign("t-dispatched", ReassignParams{DriverName: "休息乙", Reason: "x"}); err == nil {
		t.Fatal("期望失败")
	}
	var task model.FarmTask
	db.First(&task, "id = ?", "t-dispatched")
	if task.AssignedMachine != "NJ-2" || task.AssignedDriver != "在岗甲" {
		t.Errorf("任务状态被改动: %+v", task)
	}
}

func TestIntegration_LatestRecordsByTask(t *testing.T) {
	repo, db := newSQLiteRepo(t)
	seedDispatchFixture(t, db)
	records := []model.DispatchRecord{
		{ID: "r1", TaskID: "t-pending", Action: constants.DispatchActionConfirm, MachineCode: "NJ-1", Reason: "第一次"},
		{ID: "r2", TaskID: "t-pending", Action: constants.DispatchActionReassign, MachineCode: "NJ-2", Reason: "第二次"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatal(err)
	}
	got, err := repo.LatestRecordsByTask([]string{"t-pending", "t-dispatched"})
	if err != nil {
		t.Fatal(err)
	}
	if got["t-pending"] == nil || got["t-pending"].Reason != "第二次" {
		t.Errorf("latest = %+v", got["t-pending"])
	}
}

func assertNoStateChange(t *testing.T, db *gorm.DB) {
	t.Helper()
	var task model.FarmTask
	db.First(&task, "id = ?", "t-pending")
	if task.Status != constants.TaskPending || task.AssignedMachine != "" || task.AssignedDriver != "" {
		t.Errorf("失败后任务状态被改动: %+v", task)
	}
	var m1 model.Machine
	db.First(&m1, "code = ?", "NJ-1")
	if m1.Status != constants.MachineIdle {
		t.Errorf("失败后 NJ-1 状态 = %s", m1.Status)
	}
	var d1 model.Driver
	db.First(&d1, "name = ?", "在岗甲")
	if d1.Status != constants.DriverOnDuty {
		t.Errorf("失败后在岗甲状态 = %s", d1.Status)
	}
	var count int64
	db.Model(&model.DispatchRecord{}).Count(&count)
	if count != 0 {
		t.Errorf("失败后写入了 %d 条记录", count)
	}
}

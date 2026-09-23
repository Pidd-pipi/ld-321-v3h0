package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newMockRepo 构造基于 sqlmock 的派单仓储（无需真实 MySQL）。
func newMockRepo(t *testing.T) (*DispatchRepository, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	return NewDispatchRepository(gdb), mock, sqlDB
}

// expectCommit 匹配事务提交（在所有语句期望之后声明）。
func expectCommit(mock sqlmock.Sqlmock) {
	mock.ExpectCommit()
}

func TestConfirmDispatch_UsesRecommendedAndCommits(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	// 锁定并查到待派单任务
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "recommended_machine", "recommended_driver", "type", "field"}).
			AddRow("t2", "待派单", "NJ-2026-002", "何燕", "播种", "西坡旱地"))
	// 锁定并查到空闲农机
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m2", "NJ-2026-002", "空闲"))
	// 锁定并查到可派单驾驶员
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `drivers`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("d2", "何燕", "可派单"))
	// 任务、农机、驾驶员更新与记录插入
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `farm_tasks` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `machines` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `drivers` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `dispatch_records`")).WillReturnResult(sqlmock.NewResult(1, 1))
	expectCommit(mock)

	machine, driver, err := repo.ConfirmDispatch("t2", DispatchParams{Reason: ""})
	if err != nil {
		t.Fatalf("ConfirmDispatch() err = %v", err)
	}
	if machine != "NJ-2026-002" || driver != "何燕" {
		t.Errorf("ConfirmDispatch() = (%s,%s)", machine, driver)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestConfirmDispatch_TaskNotPending_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("t1", "已派单"))
	mock.ExpectRollback()

	_, _, err := repo.ConfirmDispatch("t1", DispatchParams{})
	if err == nil {
		t.Fatal("期望状态冲突错误，实际为 nil")
	}
	var conflict interface{ Error() string }
	if !errors.As(err, &conflict) {
		t.Errorf("err = %T, 期望状态冲突错误", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestConfirmDispatch_MachineBusy_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "recommended_machine", "recommended_driver"}).
			AddRow("t2", "待派单", "NJ-2026-001", "何燕"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m1", "NJ-2026-001", "作业中"))
	mock.ExpectRollback()

	_, _, err := repo.ConfirmDispatch("t2", DispatchParams{})
	if err == nil {
		t.Fatal("期望农机非空闲错误，实际为 nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestConfirmDispatch_DriverResting_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "recommended_machine", "recommended_driver"}).
			AddRow("t3", "待派单", "NJ-2026-002", "刘强"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m2", "NJ-2026-002", "空闲"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `drivers`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("d3", "刘强", "休息"))
	mock.ExpectRollback()

	_, _, err := repo.ConfirmDispatch("t3", DispatchParams{})
	if err == nil {
		t.Fatal("期望驾驶员不可派单错误，实际为 nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestConfirmDispatch_TaskNotFound_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectRollback()

	_, _, err := repo.ConfirmDispatch("missing", DispatchParams{})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, 期望 ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestReassign_ReleasesOldAndLocksNew(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	// 已派单任务：旧资源 NJ-001/周明
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assigned_machine", "assigned_driver", "type", "field"}).
			AddRow("t1", "已派单", "NJ-2026-001", "周明", "耕地", "北岭 1 号田"))
	// 按排序锁定两台农机
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines` WHERE code IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m1", "NJ-2026-001", "作业中").
			AddRow("m2", "NJ-2026-002", "空闲"))
	// 按排序锁定两名驾驶员
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `drivers` WHERE name IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("d2", "何燕", "可派单").
			AddRow("d1", "周明", "作业中"))
	// 释放旧农机、旧驾驶员；分配新农机、新驾驶员；更新任务；插入改派记录
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `machines` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `drivers` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `machines` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `drivers` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `farm_tasks` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `dispatch_records`")).WillReturnResult(sqlmock.NewResult(1, 1))
	expectCommit(mock)

	machine, driver, err := repo.Reassign("t1", ReassignParams{MachineCode: "NJ-2026-002", DriverName: "何燕", Reason: "原农机保养"})
	if err != nil {
		t.Fatalf("Reassign() err = %v", err)
	}
	if machine != "NJ-2026-002" || driver != "何燕" {
		t.Errorf("Reassign() = (%s,%s)", machine, driver)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestReassign_SameResource_Conflict(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assigned_machine", "assigned_driver"}).
			AddRow("t1", "已派单", "NJ-2026-001", "周明"))
	mock.ExpectRollback()

	_, _, err := repo.Reassign("t1", ReassignParams{MachineCode: "NJ-2026-001", DriverName: "周明", Reason: "x"})
	if err == nil {
		t.Fatal("期望新旧资源相同冲突错误")
	}
}

func TestReassign_MachineOnly_KeepsWorkingDriver(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	// 已派单任务，仅更换农机；驾驶员周明保持不变（当前作业中，不应被校验/重复更新）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assigned_machine", "assigned_driver", "type", "field"}).
			AddRow("t1", "已派单", "NJ-2026-001", "周明", "耕地", "北岭 1 号田"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines` WHERE code IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m1", "NJ-2026-001", "作业中").
			AddRow("m2", "NJ-2026-002", "空闲"))
	// 驾驶员集合只有一人（新旧相同）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `drivers` WHERE name IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("d1", "周明", "作业中"))
	// 释放旧农机 + 新机转作业中 + 更新任务 + 插入记录；驾驶员无 UPDATE
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `machines` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `machines` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `farm_tasks` SET")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `dispatch_records`")).WillReturnResult(sqlmock.NewResult(1, 1))
	expectCommit(mock)

	machine, driver, err := repo.Reassign("t1", ReassignParams{MachineCode: "NJ-2026-002", Reason: "原农机保养"})
	if err != nil {
		t.Fatalf("Reassign() err = %v", err)
	}
	if machine != "NJ-2026-002" || driver != "周明" {
		t.Errorf("Reassign() = (%s,%s), want (NJ-2026-002,周明)", machine, driver)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestReassign_NewMachineBusy_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assigned_machine", "assigned_driver", "type", "field"}).
			AddRow("t1", "已派单", "NJ-2026-001", "周明", "耕地", "北岭 1 号田"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `machines` WHERE code IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "status"}).
			AddRow("m1", "NJ-2026-001", "作业中").
			AddRow("m2", "NJ-2026-002", "作业中"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `drivers` WHERE name IN")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status"}).
			AddRow("d1", "周明", "作业中"))
	mock.ExpectRollback()

	_, _, err := repo.Reassign("t1", ReassignParams{MachineCode: "NJ-2026-002", Reason: "调整"})
	if err == nil {
		t.Fatal("期望新农机非空闲错误")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestReassign_TaskPending_Rollback(t *testing.T) {
	repo, mock, sqlDB := newMockRepo(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `farm_tasks`")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("t2", "待派单"))
	mock.ExpectRollback()

	_, _, err := repo.Reassign("t2", ReassignParams{MachineCode: "NJ-2026-005", Reason: "调整"})
	if err == nil {
		t.Fatal("期望任务非已派单错误")
	}
}

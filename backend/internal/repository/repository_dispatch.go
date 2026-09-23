package repository

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	bizerrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// clauseLockingUpdate 行级写锁（MySQL SELECT ... FOR UPDATE）。
var clauseLockingUpdate = clause.Locking{Strength: "UPDATE"}

// forUpdate 仅在 MySQL 方言下追加行级写锁；其他方言（如测试用 SQLite）原样返回。
func forUpdate(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "mysql" {
		return tx.Clauses(clauseLockingUpdate)
	}
	return tx
}

// DispatchParams 派单入参：空串表示沿用任务推荐资源。
type DispatchParams struct {
	MachineCode string
	DriverName  string
	Reason      string
}

// ReassignParams 改派入参：空串表示沿用当前派单资源。
type ReassignParams struct {
	MachineCode string
	DriverName  string
	Reason      string
}

// DispatchRepository 派单/改派数据访问。
type DispatchRepository struct {
	db *gorm.DB
}

func NewDispatchRepository(db *gorm.DB) *DispatchRepository {
	return &DispatchRepository{db: db}
}

// ConfirmDispatch 派单确认：同一事务校验并更新任务、农机、驾驶员三方状态，保存派单原因。
// 任一条件不满足则返回错误，事务回滚，任务与资源状态均不变。
// 返回最终派单的农机编号与驾驶员姓名。
func (r *DispatchRepository) ConfirmDispatch(taskID string, params DispatchParams) (string, string, error) {
	var finalMachine, finalDriver string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		task, err := lockTask(tx, taskID)
		if err != nil {
			return err
		}
		if err := validateDispatchTask(task); err != nil {
			return err
		}
		machineCode, driverName := dispatchTarget(params.MachineCode, params.DriverName, task)
		machine, err := lockMachineByCode(tx, machineCode)
		if err != nil {
			return mapMachineErr(err, machineCode)
		}
		if err := validateDispatchMachine(machine); err != nil {
			return err
		}
		driver, err := lockDriverByName(tx, driverName)
		if err != nil {
			return mapDriverErr(err, driverName)
		}
		if err := validateDispatchDriver(driver); err != nil {
			return err
		}

		now := time.Now()
		task.Status = constants.TaskDispatched
		task.AssignedMachine = machine.Code
		task.AssignedDriver = driver.Name
		if err := tx.Save(task).Error; err != nil {
			return fmt.Errorf("update task %s: %w", taskID, err)
		}

		machine.Status = constants.MachineWorking
		machine.CurrentTask = fmt.Sprintf("%s %s", task.Type, task.Field)
		if err := tx.Save(machine).Error; err != nil {
			return fmt.Errorf("update machine %s: %w", machine.Code, err)
		}

		driver.Status = constants.DriverWorking
		if err := tx.Save(driver).Error; err != nil {
			return fmt.Errorf("update driver %s: %w", driver.Name, err)
		}

		record := &model.DispatchRecord{
			ID:          uuid.NewString(),
			TaskID:      taskID,
			Action:      constants.DispatchActionConfirm,
			MachineCode: machine.Code,
			DriverName:  driver.Name,
			Reason:      params.Reason,
			CreatedAt:   now,
		}
		if err := createRecord(tx, record); err != nil {
			return err
		}
		finalMachine, finalDriver = machine.Code, driver.Name
		return nil
	})
	if err != nil {
		return "", "", err
	}
	return finalMachine, finalDriver, nil
}

// Reassign 改派：释放旧资源并在同一事务原子分配新资源。
// 任一条件不满足则返回错误，事务回滚，任务与资源状态均不变。
// 返回改派后的农机编号与驾驶员姓名。
func (r *DispatchRepository) Reassign(taskID string, params ReassignParams) (string, string, error) {
	var finalMachine, finalDriver string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		task, err := lockTask(tx, taskID)
		if err != nil {
			return err
		}
		if err := validateReassignTask(task); err != nil {
			return err
		}

		newMachineCode := firstNonEmpty(params.MachineCode, task.AssignedMachine, task.RecommendedMachine)
		newDriverName := firstNonEmpty(params.DriverName, task.AssignedDriver, task.RecommendedDriver)
		if newMachineCode == task.AssignedMachine && newDriverName == task.AssignedDriver {
			return bizConflict(constants.ErrMsgReassignSameResource)
		}

		// 按编号/姓名排序加锁，避免并发改派时交叉等待。
		machineCodes := uniqueSorted(task.AssignedMachine, newMachineCode)
		driverNames := uniqueSorted(task.AssignedDriver, newDriverName)
		machines, err := lockMachinesByCodes(tx, machineCodes)
		if err != nil {
			return err
		}
		drivers, err := lockDriversByNames(tx, driverNames)
		if err != nil {
			return err
		}

		var oldMachine, newMachine *model.Machine
		var oldDriver, newDriver *model.Driver
		for i := range machines {
			switch machines[i].Code {
			case task.AssignedMachine:
				oldMachine = &machines[i]
			}
			if machines[i].Code == newMachineCode {
				newMachine = &machines[i]
			}
		}
		for i := range drivers {
			switch drivers[i].Name {
			case task.AssignedDriver:
				oldDriver = &drivers[i]
			}
			if drivers[i].Name == newDriverName {
				newDriver = &drivers[i]
			}
		}
		if newMachine == nil {
			return mapMachineErr(gorm.ErrRecordNotFound, newMachineCode)
		}
		if newDriver == nil {
			return mapDriverErr(gorm.ErrRecordNotFound, newDriverName)
		}
		// 未变更的资源保持原派单占用，仅对新分配的资源执行空闲/可派单校验。
		if newMachine.Code != task.AssignedMachine {
			if err := validateDispatchMachine(newMachine); err != nil {
				return err
			}
		}
		if newDriver.Name != task.AssignedDriver {
			if err := validateDispatchDriver(newDriver); err != nil {
				return err
			}
		}

		now := time.Now()
		prevMachine := task.AssignedMachine
		prevDriver := task.AssignedDriver
		// 释放旧农机（仅在农机确实变更时）。
		if oldMachine != nil && oldMachine.Code != newMachine.Code {
			oldMachine.Status = constants.MachineIdle
			oldMachine.CurrentTask = ""
			if err := tx.Save(oldMachine).Error; err != nil {
				return fmt.Errorf("release machine %s: %w", oldMachine.Code, err)
			}
		}
		// 释放旧驾驶员（仅在驾驶员确实变更时）。
		if oldDriver != nil && oldDriver.Name != newDriver.Name {
			oldDriver.Status = constants.DriverAvailable
			if err := tx.Save(oldDriver).Error; err != nil {
				return fmt.Errorf("release driver %s: %w", oldDriver.Name, err)
			}
		}

		newMachine.Status = constants.MachineWorking
		newMachine.CurrentTask = fmt.Sprintf("%s %s", task.Type, task.Field)
		if err := tx.Save(newMachine).Error; err != nil {
			return fmt.Errorf("update machine %s: %w", newMachine.Code, err)
		}
		if newDriver.Name != task.AssignedDriver {
			newDriver.Status = constants.DriverWorking
			if err := tx.Save(newDriver).Error; err != nil {
				return fmt.Errorf("update driver %s: %w", newDriver.Name, err)
			}
		}

		task.AssignedMachine = newMachine.Code
		task.AssignedDriver = newDriver.Name
		if err := tx.Save(task).Error; err != nil {
			return fmt.Errorf("update task %s: %w", taskID, err)
		}

		record := &model.DispatchRecord{
			ID:          uuid.NewString(),
			TaskID:      taskID,
			Action:      constants.DispatchActionReassign,
			MachineCode: newMachine.Code,
			DriverName:  newDriver.Name,
			Reason:      params.Reason,
			PrevMachine: prevMachine,
			PrevDriver:  prevDriver,
			CreatedAt:   now,
		}
		if err := createRecord(tx, record); err != nil {
			return err
		}
		finalMachine, finalDriver = newMachine.Code, newDriver.Name
		return nil
	})
	if err != nil {
		return "", "", err
	}
	return finalMachine, finalDriver, nil
}

// LatestRecordsByTask 批量查询任务的最近一条派单/改派记录。
func (r *DispatchRepository) LatestRecordsByTask(taskIDs []string) (map[string]*model.DispatchRecord, error) {
	result := make(map[string]*model.DispatchRecord, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}
	// 按任务分区，created_at 倒序取每组最新一条（MySQL 8 ROW_NUMBER 窗口函数）。
	rankedSubQuery := r.db.Table("dispatch_records").
		Select("id, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY created_at DESC, id DESC) AS rn").
		Where("task_id IN ?", taskIDs)
	var records []model.DispatchRecord
	if err := r.db.Raw(`
		SELECT dr.id, dr.task_id, dr.action, dr.machine_code, dr.driver_name,
		       dr.reason, dr.prev_machine, dr.prev_driver, dr.created_at
		FROM dispatch_records AS dr
		JOIN (?) AS ranked ON ranked.id = dr.id AND ranked.rn = 1`, rankedSubQuery).
		Scan(&records).Error; err != nil {
		return nil, fmt.Errorf("find latest dispatch records: %w", err)
	}
	for i := range records {
		result[records[i].TaskID] = &records[i]
	}
	return result, nil
}

// lockTask 在事务内按主键锁定任务行。
func lockTask(tx *gorm.DB, taskID string) (*model.FarmTask, error) {
	var task model.FarmTask
	err := forUpdate(tx).First(&task, "id = ?", taskID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%s: %w", constants.ErrMsgTaskNotFound, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("lock task %s: %w", taskID, err)
	}
	return &task, nil
}

// lockMachineByCode 在事务内按编号锁定农机行。
func lockMachineByCode(tx *gorm.DB, code string) (*model.Machine, error) {
	var machine model.Machine
	err := forUpdate(tx).First(&machine, "code = ?", code).Error
	if err != nil {
		return nil, err
	}
	return &machine, nil
}

// lockDriverByName 在事务内按姓名锁定驾驶员行。
func lockDriverByName(tx *gorm.DB, name string) (*model.Driver, error) {
	var driver model.Driver
	err := forUpdate(tx).First(&driver, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &driver, nil
}

// lockMachinesByCodes 在事务内按编号批量锁定农机行。
func lockMachinesByCodes(tx *gorm.DB, codes []string) ([]model.Machine, error) {
	var machines []model.Machine
	if len(codes) == 0 {
		return machines, nil
	}
	if err := forUpdate(tx).Where("code IN ?", codes).Find(&machines).Error; err != nil {
		return nil, fmt.Errorf("lock machines: %w", err)
	}
	return machines, nil
}

// lockDriversByNames 在事务内按姓名批量锁定驾驶员行。
func lockDriversByNames(tx *gorm.DB, names []string) ([]model.Driver, error) {
	var drivers []model.Driver
	if len(names) == 0 {
		return drivers, nil
	}
	if err := forUpdate(tx).Where("name IN ?", names).Find(&drivers).Error; err != nil {
		return nil, fmt.Errorf("lock drivers: %w", err)
	}
	return drivers, nil
}

// createRecord 写入派单/改派记录。
func createRecord(tx *gorm.DB, record *model.DispatchRecord) error {
	if err := tx.Create(record).Error; err != nil {
		return fmt.Errorf("create dispatch record: %w", err)
	}
	return nil
}

// mapMachineErr 将农机查找错误转换为明确的业务错误信息。
func mapMachineErr(err error, code string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s(%s): %w", constants.ErrMsgMachineNotFound, code, ErrNotFound)
	}
	return fmt.Errorf("find machine %s: %w", code, err)
}

// mapDriverErr 将驾驶员查找错误转换为明确的业务错误信息。
func mapDriverErr(err error, name string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s(%s): %w", constants.ErrMsgDriverNotFound, name, ErrNotFound)
	}
	return fmt.Errorf("find driver %s: %w", name, err)
}

func bizConflict(message string) error {
	return bizerrors.NewStateConflict(message)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func uniqueSorted(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

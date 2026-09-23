package repository

import (
	"errors"
	"fmt"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"gorm.io/gorm"
)

// ErrStateConflict 资源状态已被其他事务变更（条件更新影响行数为 0）。
var ErrStateConflict = errors.New("resource state changed concurrently")

// DispatchRepository 派单数据访问。所有方法均接受事务句柄，由 service 层统一控制事务。
type DispatchRepository struct {
	db *gorm.DB
}

func NewDispatchRepository(db *gorm.DB) *DispatchRepository {
	return &DispatchRepository{db: db}
}

// InTx 在单个数据库事务内执行派单动作，返回 error 时整体回滚。
func (r *DispatchRepository) InTx(fn func(tx *gorm.DB) error) error {
	if err := r.db.Transaction(fn); err != nil {
		return fmt.Errorf("dispatch transaction: %w", err)
	}
	return nil
}

// FindTaskTx 事务内查找任务。
func (r *DispatchRepository) FindTaskTx(tx *gorm.DB, id string) (*model.FarmTask, error) {
	var t model.FarmTask
	err := tx.First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	return &t, nil
}

// FindMachineByCodeTx 事务内按编号查找农机。
func (r *DispatchRepository) FindMachineByCodeTx(tx *gorm.DB, code string) (*model.Machine, error) {
	var m model.Machine
	err := tx.First(&m, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find machine by code: %w", err)
	}
	return &m, nil
}

// FindDriverByNameTx 事务内按姓名查找驾驶员。
func (r *DispatchRepository) FindDriverByNameTx(tx *gorm.DB, name string) (*model.Driver, error) {
	var d model.Driver
	err := tx.First(&d, "name = ?", name).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find driver by name: %w", err)
	}
	return &d, nil
}

// OccupyMachineTx 条件更新：仅当农机仍为空闲时置为作业中，避免并发重复派单。
func (r *DispatchRepository) OccupyMachineTx(tx *gorm.DB, code, taskLabel string) error {
	res := tx.Model(&model.Machine{}).
		Where("code = ? AND status = ?", code, constants.MachineIdle).
		Updates(map[string]interface{}{
			"status":       constants.MachineWorking,
			"current_task": taskLabel,
		})
	if res.Error != nil {
		return fmt.Errorf("occupy machine: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrStateConflict
	}
	return nil
}

// ReleaseMachineTx 条件更新：仅当农机仍为作业中时释放为空闲。
func (r *DispatchRepository) ReleaseMachineTx(tx *gorm.DB, code string) error {
	res := tx.Model(&model.Machine{}).
		Where("code = ? AND status = ?", code, constants.MachineWorking).
		Updates(map[string]interface{}{
			"status":       constants.MachineIdle,
			"current_task": constants.IdleMachineTask,
		})
	if res.Error != nil {
		return fmt.Errorf("release machine: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrStateConflict
	}
	return nil
}

// OccupyDriverTx 条件更新：仅当驾驶员仍在岗或可派单时置为作业中。
func (r *DispatchRepository) OccupyDriverTx(tx *gorm.DB, name string) error {
	res := tx.Model(&model.Driver{}).
		Where("name = ? AND status IN ?", name, []string{constants.DriverIdle, constants.DriverDispatch}).
		Update("status", constants.DriverWorking)
	if res.Error != nil {
		return fmt.Errorf("occupy driver: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrStateConflict
	}
	return nil
}

// ReleaseDriverTx 条件更新：仅当驾驶员仍为作业中时释放为可派单。
func (r *DispatchRepository) ReleaseDriverTx(tx *gorm.DB, name string) error {
	res := tx.Model(&model.Driver{}).
		Where("name = ? AND status = ?", name, constants.DriverWorking).
		Update("status", constants.DriverDispatch)
	if res.Error != nil {
		return fmt.Errorf("release driver: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrStateConflict
	}
	return nil
}

// SaveTaskTx 事务内保存任务。
func (r *DispatchRepository) SaveTaskTx(tx *gorm.DB, t *model.FarmTask) error {
	if err := tx.Save(t).Error; err != nil {
		return fmt.Errorf("save task: %w", err)
	}
	return nil
}

// CreateDispatchRecordTx 事务内写入派单/改派记录。
func (r *DispatchRepository) CreateDispatchRecordTx(tx *gorm.DB, rec *model.DispatchRecord) error {
	if err := tx.Create(rec).Error; err != nil {
		return fmt.Errorf("create dispatch record: %w", err)
	}
	return nil
}

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/agridispatch/agridispatch/internal/constants"
	apperrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DispatchService 任务派单确认与改派服务。
type DispatchService struct {
	repo         *repository.DispatchRepository
	dashboardSvc *DashboardService
	logger       *slog.Logger
}

func NewDispatchService(repo *repository.DispatchRepository, dashboardSvc *DashboardService, logger *slog.Logger) *DispatchService {
	return &DispatchService{repo: repo, dashboardSvc: dashboardSvc, logger: logger}
}

// Confirm 派单确认：任务须待派单、农机须空闲、驾驶员须在岗或可派单；
// 校验失败返回明确业务错误且任务与资源状态均不变，成功后同一事务更新三方状态并保存原因。
func (s *DispatchService) Confirm(ctx context.Context, taskID string, machineCode, driverName, reason string) (*model.DispatchRecord, error) {
	if strings.TrimSpace(reason) == "" {
		reason = constants.DefaultDispatchReason
	}
	machineCode = strings.TrimSpace(machineCode)
	driverName = strings.TrimSpace(driverName)

	var record *model.DispatchRecord
	err := s.repo.InTx(func(tx *gorm.DB) error {
		task, err := s.repo.FindTaskTx(tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != constants.TaskPending {
			return apperrors.NewWithStatus(constants.CodeTaskNotPending, 409,
				fmt.Sprintf(constants.MsgTaskNotPending, taskID, task.Status))
		}

		if machineCode == "" {
			machineCode = strings.TrimSpace(task.RecommendedMachine)
		}
		if driverName == "" {
			driverName = strings.TrimSpace(task.RecommendedDriver)
		}
		if machineCode == "" || driverName == "" {
			return apperrors.NewWithStatus(constants.CodeBadRequest, 400,
				constants.MsgNoRecommended)
		}

		if err := s.assertMachineIdle(tx, machineCode); err != nil {
			return err
		}
		if err := s.assertDriverAvailable(tx, driverName); err != nil {
			return err
		}

		taskLabel := fmt.Sprintf("%s %s", task.Type, task.Field)
		if err := s.repo.OccupyMachineTx(tx, machineCode, taskLabel); err != nil {
			return s.wrapConflict(err, constants.CodeMachineNotIdle,
				fmt.Sprintf(constants.MsgMachineNotIdle, machineCode, constants.MachineWorking))
		}
		if err := s.repo.OccupyDriverTx(tx, driverName); err != nil {
			return s.wrapConflict(err, constants.CodeDriverUnavailable,
				fmt.Sprintf(constants.MsgDriverUnavailable, driverName, constants.DriverWorking))
		}

		task.Status = constants.TaskDispatched
		task.AssignedMachine = machineCode
		task.AssignedDriver = driverName
		task.DispatchReason = reason
		if err := s.repo.SaveTaskTx(tx, task); err != nil {
			return err
		}

		record = s.buildRecord(taskID, constants.DispatchActionConfirm, machineCode, driverName, "", "", reason)
		if err := s.repo.CreateDispatchRecordTx(tx, record); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("dispatch confirm rejected", "taskId", taskID, "err", err.Error())
		return nil, err
	}

	s.dashboardSvc.Invalidate(ctx)
	s.logger.Info("task dispatched", "taskId", taskID, "machine", machineCode, "driver", driverName)
	return record, nil
}

// Reassign 已派单任务带原因改派：释放旧资源并在同一事务原子分配新资源；
// 未传农机/驾驶员时沿用推荐资源；失败不改任何状态。
func (s *DispatchService) Reassign(ctx context.Context, taskID string, machineCode, driverName, reason string) (*model.DispatchRecord, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, apperrors.NewWithStatus(constants.CodeBadRequest, 400, constants.MsgReassignReason)
	}
	machineCode = strings.TrimSpace(machineCode)
	driverName = strings.TrimSpace(driverName)

	var record *model.DispatchRecord
	err := s.repo.InTx(func(tx *gorm.DB) error {
		task, err := s.repo.FindTaskTx(tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != constants.TaskDispatched {
			return apperrors.NewWithStatus(constants.CodeTaskNotDispatched, 409,
				fmt.Sprintf(constants.MsgTaskNotDispatched, taskID, task.Status))
		}

		oldMachine := task.AssignedMachine
		oldDriver := task.AssignedDriver
		if machineCode == "" {
			machineCode = strings.TrimSpace(task.RecommendedMachine)
		}
		if driverName == "" {
			driverName = strings.TrimSpace(task.RecommendedDriver)
		}
		if machineCode == "" || driverName == "" {
			return apperrors.NewWithStatus(constants.CodeBadRequest, 400,
				constants.MsgNoRecommended)
		}
		if machineCode == oldMachine && driverName == oldDriver {
			return apperrors.NewWithStatus(constants.CodeReassignSameTarget, 409, constants.MsgReassignSameTarget)
		}

		if machineCode != oldMachine {
			if err := s.assertMachineIdle(tx, machineCode); err != nil {
				return err
			}
		}
		if driverName != oldDriver {
			if err := s.assertDriverAvailable(tx, driverName); err != nil {
				return err
			}
		}

		if machineCode != oldMachine {
			if err := s.repo.ReleaseMachineTx(tx, oldMachine); err != nil {
				return s.wrapConflict(err, constants.CodeMachineNotIdle,
					fmt.Sprintf(constants.MsgOldMachineChanged, oldMachine))
			}
			if err := s.repo.OccupyMachineTx(tx, machineCode, fmt.Sprintf("%s %s", task.Type, task.Field)); err != nil {
				return s.wrapConflict(err, constants.CodeMachineNotIdle,
					fmt.Sprintf(constants.MsgMachineNotIdle, machineCode, constants.MachineWorking))
			}
		}
		if driverName != oldDriver {
			if err := s.repo.ReleaseDriverTx(tx, oldDriver); err != nil {
				return s.wrapConflict(err, constants.CodeDriverUnavailable,
					fmt.Sprintf(constants.MsgOldDriverChanged, oldDriver))
			}
			if err := s.repo.OccupyDriverTx(tx, driverName); err != nil {
				return s.wrapConflict(err, constants.CodeDriverUnavailable,
					fmt.Sprintf(constants.MsgDriverUnavailable, driverName, constants.DriverWorking))
			}
		}

		task.AssignedMachine = machineCode
		task.AssignedDriver = driverName
		task.DispatchReason = reason
		if err := s.repo.SaveTaskTx(tx, task); err != nil {
			return err
		}

		record = s.buildRecord(taskID, constants.DispatchActionReassign, machineCode, driverName, oldMachine, oldDriver, reason)
		if err := s.repo.CreateDispatchRecordTx(tx, record); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("dispatch reassign rejected", "taskId", taskID, "err", err.Error())
		return nil, err
	}

	s.dashboardSvc.Invalidate(ctx)
	s.logger.Info("task reassigned", "taskId", taskID,
		"oldMachine", record.PreviousMachine, "oldDriver", record.PreviousDriver,
		"machine", machineCode, "driver", driverName)
	return record, nil
}

// assertMachineIdle 校验农机存在且空闲。
func (s *DispatchService) assertMachineIdle(tx *gorm.DB, code string) error {
	machine, err := s.repo.FindMachineByCodeTx(tx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperrors.NewWithStatus(constants.CodeMachineNotFound, 404,
				fmt.Sprintf(constants.MsgMachineNotFound, code))
		}
		return err
	}
	if machine.Status != constants.MachineIdle {
		return apperrors.NewWithStatus(constants.CodeMachineNotIdle, 409,
			fmt.Sprintf(constants.MsgMachineNotIdle, code, machine.Status))
	}
	return nil
}

// assertDriverAvailable 校验驾驶员存在且在岗或可派单。
func (s *DispatchService) assertDriverAvailable(tx *gorm.DB, name string) error {
	driver, err := s.repo.FindDriverByNameTx(tx, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperrors.NewWithStatus(constants.CodeDriverNotFound, 404,
				fmt.Sprintf(constants.MsgDriverNotFound, name))
		}
		return err
	}
	if driver.Status != constants.DriverIdle && driver.Status != constants.DriverDispatch {
		return apperrors.NewWithStatus(constants.CodeDriverUnavailable, 409,
			fmt.Sprintf(constants.MsgDriverUnavailable, name, driver.Status))
	}
	return nil
}

// wrapConflict 将条件更新失败（并发冲突）转成业务错误。
func (s *DispatchService) wrapConflict(err error, code int, message string) error {
	if errors.Is(err, repository.ErrStateConflict) {
		return apperrors.NewWithStatus(code, 409, message)
	}
	return err
}

// buildRecord 组装派单/改派记录。
func (s *DispatchService) buildRecord(taskID, action, machine, driver, oldMachine, oldDriver, reason string) *model.DispatchRecord {
	return &model.DispatchRecord{
		ID:              "dr-" + uuid.NewString(),
		TaskID:          taskID,
		Action:          action,
		MachineCode:     machine,
		DriverName:      driver,
		PreviousMachine: oldMachine,
		PreviousDriver:  oldDriver,
		Reason:          reason,
		Operator:        constants.DefaultDispatchOperator,
	}
}

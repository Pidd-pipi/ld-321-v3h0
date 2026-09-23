package repository

import (
	"fmt"

	"github.com/agridispatch/agridispatch/internal/constants"
	bizerrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
)

// dispatchTarget 解析实际派单资源：请求指定则用指定资源，未传沿用任务推荐资源。
func dispatchTarget(machineCode, driverName string, task *model.FarmTask) (string, string) {
	if machineCode == "" {
		machineCode = task.RecommendedMachine
	}
	if driverName == "" {
		driverName = task.RecommendedDriver
	}
	return machineCode, driverName
}

// validateDispatchTask 派单前置校验：任务必须处于待派单状态。
func validateDispatchTask(task *model.FarmTask) error {
	if task.Status != constants.TaskPending {
		return bizerrors.NewStateConflict(fmt.Sprintf(constants.ErrMsgTaskNotPending, task.Status))
	}
	return nil
}

// validateReassignTask 改派前置校验：任务必须处于已派单状态。
func validateReassignTask(task *model.FarmTask) error {
	if task.Status != constants.TaskDispatched {
		return bizerrors.NewStateConflict(fmt.Sprintf(constants.ErrMsgTaskNotDispatched, task.Status))
	}
	return nil
}

// validateDispatchMachine 资源校验：农机必须空闲。
func validateDispatchMachine(m *model.Machine) error {
	if m.Status != constants.MachineIdle {
		return bizerrors.NewStateConflict(fmt.Sprintf(constants.ErrMsgMachineNotIdle, m.Code, m.Status))
	}
	return nil
}

// validateDispatchDriver 资源校验：驾驶员必须在岗或可派单。
func validateDispatchDriver(d *model.Driver) error {
	if d.Status != constants.DriverOnDuty && d.Status != constants.DriverAvailable {
		return bizerrors.NewStateConflict(fmt.Sprintf(constants.ErrMsgDriverNotDispatchable, d.Name, d.Status))
	}
	return nil
}

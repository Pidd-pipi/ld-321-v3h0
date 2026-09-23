package repository

import (
	"errors"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	bizerrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/model"
)

func TestDispatchTarget(t *testing.T) {
	task := &model.FarmTask{RecommendedMachine: "NJ-2026-002", RecommendedDriver: "何燕"}
	tests := []struct {
		name        string
		machineCode string
		driverName  string
		wantMachine string
		wantDriver  string
	}{
		{name: "全部沿用推荐", machineCode: "", driverName: "", wantMachine: "NJ-2026-002", wantDriver: "何燕"},
		{name: "仅指定农机", machineCode: "NJ-2026-005", driverName: "", wantMachine: "NJ-2026-005", wantDriver: "何燕"},
		{name: "仅指定驾驶员", machineCode: "", driverName: "周明", wantMachine: "NJ-2026-002", wantDriver: "周明"},
		{name: "全部指定", machineCode: "NJ-2026-005", driverName: "周明", wantMachine: "NJ-2026-005", wantDriver: "周明"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMachine, gotDriver := dispatchTarget(tt.machineCode, tt.driverName, task)
			if gotMachine != tt.wantMachine || gotDriver != tt.wantDriver {
				t.Errorf("dispatchTarget() = (%s,%s), want (%s,%s)", gotMachine, gotDriver, tt.wantMachine, tt.wantDriver)
			}
		})
	}
}

func TestValidateDispatchTask(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{status: constants.TaskPending, wantErr: false},
		{status: constants.TaskDispatched, wantErr: true},
		{status: constants.TaskDone, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := validateDispatchTask(&model.FarmTask{Status: tt.status})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDispatchTask() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.As(err, new(*bizerrors.StateConflictError)) {
				t.Errorf("validateDispatchTask() err type = %T, want *StateConflictError", err)
			}
		})
	}
}

func TestValidateReassignTask(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{status: constants.TaskDispatched, wantErr: false},
		{status: constants.TaskPending, wantErr: true},
		{status: constants.TaskDone, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := validateReassignTask(&model.FarmTask{Status: tt.status})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateReassignTask() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDispatchMachine(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{status: constants.MachineIdle, wantErr: false},
		{status: constants.MachineWorking, wantErr: true},
		{status: constants.MachineRepair, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := validateDispatchMachine(&model.Machine{Code: "NJ-1", Status: tt.status})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDispatchMachine() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() == "" {
				t.Errorf("冲突错误必须包含明确信息")
			}
		})
	}
}

func TestValidateDispatchDriver(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{status: constants.DriverOnDuty, wantErr: false},
		{status: constants.DriverAvailable, wantErr: false},
		{status: constants.DriverWorking, wantErr: true},
		{status: constants.DriverResting, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := validateDispatchDriver(&model.Driver{Name: "何燕", Status: tt.status})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDispatchDriver() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUniqueSorted(t *testing.T) {
	got := uniqueSorted("NJ-3", "NJ-1", "NJ-1", "", "NJ-2")
	want := []string{"NJ-1", "NJ-2", "NJ-3"}
	if len(got) != len(want) {
		t.Fatalf("uniqueSorted() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniqueSorted()[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x"); got != "x" {
		t.Errorf("firstNonEmpty() = %s, want x", got)
	}
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("firstNonEmpty() = %s, want a", got)
	}
}

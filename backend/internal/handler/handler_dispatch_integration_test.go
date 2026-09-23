//go:build integration

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/handler"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupDispatchRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FarmTask{}, &model.Machine{}, &model.Driver{}, &model.DispatchRecord{}); err != nil {
		t.Fatal(err)
	}
	fixtures := []interface{}{
		&model.Machine{ID: "m1", Code: "NJ-IDLE", Status: constants.MachineIdle},
		&model.Machine{ID: "m2", Code: "NJ-BUSY", Status: constants.MachineWorking, CurrentTask: "其他任务"},
		&model.Driver{ID: "d1", Name: "在岗甲", Status: constants.DriverOnDuty},
		&model.Driver{ID: "d2", Name: "休息乙", Status: constants.DriverResting},
		&model.FarmTask{ID: "t1", Type: "耕地", Field: "北田", Status: constants.TaskPending, RecommendedMachine: "NJ-IDLE", RecommendedDriver: "在岗甲"},
		&model.FarmTask{ID: "t2", Type: "播种", Field: "南田", Status: constants.TaskDispatched, AssignedMachine: "NJ-BUSY", AssignedDriver: "在岗甲", RecommendedMachine: "NJ-BUSY", RecommendedDriver: "在岗甲"},
	}
	for _, f := range fixtures {
		if err := db.Create(f).Error; err != nil {
			t.Fatal(err)
		}
	}
	dispatchSvc := service.NewDispatchService(repository.NewDispatchRepository(db), nil, nil)
	h := handler.NewDispatchHandler(dispatchSvc)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.POST("/tasks/:id/dispatch", h.ConfirmDispatch)
	v1.POST("/tasks/:id/reassign", h.Reassign)
	return r, db
}

func perform(t *testing.T, r *gin.Engine, method, path string, body interface{}) (int, envelope) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	return w.Code, env
}

func TestHTTP_DispatchWithRecommended_Succeeds(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, env := perform(t, r, http.MethodPost, "/api/v1/tasks/t1/dispatch", map[string]interface{}{})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("status=%d env=%+v", status, env)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["machineCode"] != "NJ-IDLE" || data["driverName"] != "在岗甲" || data["status"] != constants.TaskDispatched {
		t.Errorf("data = %+v", data)
	}
}

func TestHTTP_DispatchBusyMachine_ConflictNoStateChange(t *testing.T) {
	r, db := setupDispatchRouter(t)
	status, env := perform(t, r, http.MethodPost, "/api/v1/tasks/t1/dispatch",
		map[string]string{"machineCode": "NJ-BUSY"})
	if status != http.StatusConflict || env.Code != constants.CodeDispatchConflict {
		t.Fatalf("status=%d env=%+v", status, env)
	}
	if env.Message == "" {
		t.Error("冲突错误必须返回明确信息")
	}
	var task model.FarmTask
	db.First(&task, "id = ?", "t1")
	if task.Status != constants.TaskPending || task.AssignedMachine != "" {
		t.Errorf("失败后任务状态被改动: %+v", task)
	}
}

func TestHTTP_DispatchRestingDriver_Conflict(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, env := perform(t, r, http.MethodPost, "/api/v1/tasks/t1/dispatch",
		map[string]string{"driverName": "休息乙"})
	if status != http.StatusConflict {
		t.Fatalf("status=%d env=%+v", status, env)
	}
}

func TestHTTP_DispatchAlreadyDispatched_Conflict(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, _ := perform(t, r, http.MethodPost, "/api/v1/tasks/t2/dispatch", map[string]interface{}{})
	if status != http.StatusConflict {
		t.Fatalf("status=%d, want 409", status)
	}
}

func TestHTTP_DispatchMissingTask_NotFound(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, _ := perform(t, r, http.MethodPost, "/api/v1/tasks/nope/dispatch", map[string]interface{}{})
	if status != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", status)
	}
}

func TestHTTP_ReassignWithoutReason_BadRequest(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, _ := perform(t, r, http.MethodPost, "/api/v1/tasks/t2/reassign",
		map[string]string{"machineCode": "NJ-IDLE", "reason": "   "})
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", status)
	}
}

func TestHTTP_ReassignPendingTask_Conflict(t *testing.T) {
	r, _ := setupDispatchRouter(t)
	status, _ := perform(t, r, http.MethodPost, "/api/v1/tasks/t1/reassign",
		map[string]string{"machineCode": "NJ-IDLE", "reason": "调整"})
	if status != http.StatusConflict {
		t.Fatalf("status=%d, want 409", status)
	}
}

func TestHTTP_DispatchThenReassign_FullFlow(t *testing.T) {
	r, db := setupDispatchRouter(t)
	// t2 已派给 NJ-BUSY/在岗甲；先派 t1 占用 NJ-IDLE 不合适。改派 t2 到一台新空闲农机 NJ-IDLE2。
	if err := db.Create(&model.Machine{ID: "m3", Code: "NJ-IDLE2", Status: constants.MachineIdle}).Error; err != nil {
		t.Fatal(err)
	}
	status, env := perform(t, r, http.MethodPost, "/api/v1/tasks/t2/reassign",
		map[string]string{"machineCode": "NJ-IDLE2", "reason": "原农机保养"})
	if status != http.StatusOK {
		t.Fatalf("reassign status=%d env=%+v", status, env)
	}
	var oldM model.Machine
	db.First(&oldM, "code = ?", "NJ-BUSY")
	if oldM.Status != constants.MachineIdle {
		t.Errorf("旧农机未释放: %s", oldM.Status)
	}
	var newM model.Machine
	db.First(&newM, "code = ?", "NJ-IDLE2")
	if newM.Status != constants.MachineWorking {
		t.Errorf("新农机未占用: %s", newM.Status)
	}
	var task model.FarmTask
	db.First(&task, "id = ?", "t2")
	if task.AssignedMachine != "NJ-IDLE2" {
		t.Errorf("任务资源未更新: %+v", task)
	}
	var rec model.DispatchRecord
	if err := db.Where("task_id = ? AND action = ?", "t2", constants.DispatchActionReassign).First(&rec).Error; err != nil {
		t.Fatalf("改派记录未保存: %v", err)
	}
	if rec.PrevMachine != "NJ-BUSY" || rec.Reason != "原农机保养" {
		t.Errorf("改派记录 = %+v", rec)
	}
}

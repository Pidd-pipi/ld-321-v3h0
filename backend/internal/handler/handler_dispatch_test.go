package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func setupDispatchRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:handler-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Machine{}, &model.FarmTask{}, &model.Driver{}, &model.DispatchRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&[]model.Machine{
		{ID: "m2", Code: "NJ-2026-002", Status: constants.MachineIdle, CurrentTask: constants.IdleMachineTask},
		{ID: "m3", Code: "NJ-2026-003", Status: constants.MachineRepair},
	})
	db.Create(&[]model.Driver{
		{ID: "d2", Name: "何燕", Status: constants.DriverDispatch},
		{ID: "d3", Name: "刘强", Status: constants.DriverResting},
	})
	db.Create(&model.FarmTask{ID: "t1", Type: "播种", Field: "西坡旱地", Status: constants.TaskPending, RecommendedMachine: "NJ-2026-002", RecommendedDriver: "何燕"})

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = rdb.Close() })
	logger := slog.Default()
	dashSvc := service.NewDashboardService(repository.NewDashboardRepository(db), rdb, logger)
	dispatchSvc := service.NewDispatchService(repository.NewDispatchRepository(db), dashSvc, logger)

	r := gin.New()
	h := NewDispatchHandler(dispatchSvc)
	v1 := r.Group("/api/v1")
	v1.POST("/tasks/:id/dispatch", h.Confirm)
	v1.POST("/tasks/:id/reassign", h.Reassign)
	return r, db
}

func postJSON(t *testing.T, r *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(http.MethodPost, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestDispatchConfirmHTTP(t *testing.T) {
	t.Run("空请求体也可派单并沿用推荐资源", func(t *testing.T) {
		r, db := setupDispatchRouter(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/t1/dispatch", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
		}
		var resp apiResp
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Code != constants.CodeOK {
			t.Fatalf("response = %s", w.Body.String())
		}
		var task model.FarmTask
		db.First(&task, "id = ?", "t1")
		if task.Status != constants.TaskDispatched || task.AssignedMachine != "NJ-2026-002" || task.AssignedDriver != "何燕" {
			t.Fatalf("task = %+v", task)
		}
	})

	t.Run("重复派单返回 409 与明确错误码", func(t *testing.T) {
		r, _ := setupDispatchRouter(t)
		w := postJSON(t, r, "/api/v1/tasks/t1/dispatch", map[string]string{"reason": "首次"})
		if w.Code != http.StatusOK {
			t.Fatalf("first dispatch status = %d", w.Code)
		}
		w = postJSON(t, r, "/api/v1/tasks/t1/dispatch", map[string]string{"reason": "再次"})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d want 409, body=%s", w.Code, w.Body.String())
		}
		var resp apiResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != constants.CodeTaskNotPending {
			t.Fatalf("code = %d want %d msg=%s", resp.Code, constants.CodeTaskNotPending, resp.Message)
		}
	})

	t.Run("维修农机派单返回 409", func(t *testing.T) {
		r, _ := setupDispatchRouter(t)
		w := postJSON(t, r, "/api/v1/tasks/t1/dispatch", map[string]string{"machineCode": "NJ-2026-003", "driverName": "何燕"})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d want 409", w.Code)
		}
	})

	t.Run("不存在农机返回 404", func(t *testing.T) {
		r, _ := setupDispatchRouter(t)
		w := postJSON(t, r, "/api/v1/tasks/t1/dispatch", map[string]string{"machineCode": "NJ-X", "driverName": "何燕"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d want 404 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("待派单任务改派返回 409", func(t *testing.T) {
		r, _ := setupDispatchRouter(t)
		w := postJSON(t, r, "/api/v1/tasks/t1/reassign", map[string]string{"reason": "先改派"})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d want 409", w.Code)
		}
	})

	t.Run("改派缺少原因返回 400", func(t *testing.T) {
		r, db := setupDispatchRouter(t)
		postJSON(t, r, "/api/v1/tasks/t1/dispatch", nil)
		w := postJSON(t, r, "/api/v1/tasks/t1/reassign", map[string]string{"machineCode": "NJ-2026-002"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d want 400", w.Code)
		}
		var task model.FarmTask
		db.First(&task, "id = ?", "t1")
		if task.DispatchReason == "" || task.AssignedDriver != "何燕" {
			t.Fatalf("task unexpectedly changed: %+v", task)
		}
	})
}

package handler

import (
	"net/http"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/dto"
	apperrors "github.com/agridispatch/agridispatch/internal/errors"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/util"
	"github.com/gin-gonic/gin"
)

// DispatchHandler 任务派单确认与改派处理器。
type DispatchHandler struct {
	dispatchSvc *service.DispatchService
}

func NewDispatchHandler(dispatchSvc *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{dispatchSvc: dispatchSvc}
}

// Confirm 派单确认（请求可指定农机和驾驶员，未传时沿用推荐资源）。
func (h *DispatchHandler) Confirm(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "task id required")
		return
	}
	var req dto.DispatchConfirmRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
			return
		}
	}
	record, err := h.dispatchSvc.Confirm(c.Request.Context(), taskID, req.MachineCode, req.DriverName, req.Reason)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, dto.DispatchResponse{
		TaskID:      record.TaskID,
		Status:      constants.TaskDispatched,
		MachineCode: record.MachineCode,
		DriverName:  record.DriverName,
		Reason:      record.Reason,
		Action:      record.Action,
		Message:     "派单成功",
	})
}

// Reassign 已派单任务带原因改派（释放旧资源并原子分配新资源）。
func (h *DispatchHandler) Reassign(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "task id required")
		return
	}
	var req dto.DispatchReassignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
		return
	}
	if req.Reason == "" {
		util.FailError(c, apperrors.NewWithStatus(constants.CodeBadRequest, http.StatusBadRequest, constants.MsgReassignReason))
		return
	}
	record, err := h.dispatchSvc.Reassign(c.Request.Context(), taskID, req.MachineCode, req.DriverName, req.Reason)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, dto.DispatchResponse{
		TaskID:      record.TaskID,
		Status:      constants.TaskDispatched,
		MachineCode: record.MachineCode,
		DriverName:  record.DriverName,
		Reason:      record.Reason,
		Action:      record.Action,
		Message:     "改派成功",
	})
}

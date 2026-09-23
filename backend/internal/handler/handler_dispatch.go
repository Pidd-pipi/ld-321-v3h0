package handler

import (
	"net/http"

	"github.com/agridispatch/agridispatch/internal/dto"
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

// ConfirmDispatch 派单确认（可指定农机/驾驶员，未传沿用推荐资源）。
func (h *DispatchHandler) ConfirmDispatch(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, 40000, "task id required")
		return
	}
	var req dto.DispatchConfirmRequest
	// 允许空请求体（一键派单场景全部沿用推荐资源）。
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			util.Fail(c, http.StatusBadRequest, 40000, "请求参数格式有误")
			return
		}
	}
	res, err := h.dispatchSvc.ConfirmDispatch(c.Request.Context(), taskID, req)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, res)
}

// Reassign 已派单任务带原因改派。
func (h *DispatchHandler) Reassign(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		util.Fail(c, http.StatusBadRequest, 40000, "task id required")
		return
	}
	var req dto.ReassignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, 40000, "改派原因不能为空")
		return
	}
	res, err := h.dispatchSvc.Reassign(c.Request.Context(), taskID, req)
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, res)
}

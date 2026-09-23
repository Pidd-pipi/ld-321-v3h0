package handler

import (
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/util"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 调度看板处理器。
type DashboardHandler struct {
	dashboardSvc *service.DashboardService
}

func NewDashboardHandler(dashboardSvc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc}
}

// Overview 看板总览。
func (h *DashboardHandler) Overview(c *gin.Context) {
	ov, err := h.dashboardSvc.Overview(c.Request.Context())
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, ov)
}

// ExportReport 作业报表导出。
func (h *DashboardHandler) ExportReport(c *gin.Context) {
	res, err := h.dashboardSvc.ExportReport(c.Request.Context())
	if err != nil {
		util.FailError(c, err)
		return
	}
	util.OK(c, res)
}

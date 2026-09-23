package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const swaggerJSON = `{
  "swagger": "2.0",
  "info": {
    "title": "农机调度管理系统 API",
    "description": "农机资源管理、作业任务调度、实时轨迹监控、作业统计与维修保养提醒。",
    "version": "1.0.0"
  },
  "basePath": "/api/v1",
  "schemes": ["http", "ws"],
  "paths": {
    "/auth/login": { "post": { "summary": "登录", "tags": ["auth"] } },
    "/auth/me": { "get": { "summary": "当前用户", "tags": ["auth"] } },
    "/dashboard/overview": { "get": { "summary": "调度看板总览", "tags": ["dashboard"] } },
    "/tasks/{id}/dispatch": {
      "post": {
        "summary": "派单确认",
        "description": "请求体可指定 machineCode/driverName，未传时沿用任务推荐资源；reason 可空。校验任务待派单、农机空闲、驾驶员在岗或可派单，成功后同一事务更新任务/农机/驾驶员三方状态并保存派单记录。",
        "tags": ["tasks"],
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "string" }],
        "responses": {
          "200": { "description": "派单成功" },
          "404": { "description": "任务/农机/驾驶员不存在" },
          "409": { "description": "任务非待派单、农机非空闲或驾驶员不可派单，状态不变" }
        }
      }
    },
    "/tasks/{id}/reassign": {
      "post": {
        "summary": "已派单任务改派",
        "description": "请求体可指定新的 machineCode/driverName（未传沿用推荐资源），reason 必填。同一事务释放旧农机/旧驾驶员并原子分配新资源，写入改派记录；任一步失败整体回滚，任务与资源状态不变。",
        "tags": ["tasks"],
        "parameters": [{ "name": "id", "in": "path", "required": true, "type": "string" }],
        "responses": {
          "200": { "description": "改派成功" },
          "400": { "description": "缺少改派原因" },
          "404": { "description": "任务/农机/驾驶员不存在" },
          "409": { "description": "任务非已派单、新资源不可用或新旧资源一致，状态不变" }
        }
      }
    },
    "/dashboard/reports/work/export": { "get": { "summary": "作业报表导出", "tags": ["dashboard"] } }
  }
}`

// SwaggerJSON 提供 swagger.json。
func (h *HealthHandler) SwaggerJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, swaggerJSON)
}

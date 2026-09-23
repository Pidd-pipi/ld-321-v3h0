package handler

import (
	"context"
	"time"

	"github.com/agridispatch/agridispatch/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler 健康检查。
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Healthz 存活检查。
func (h *HealthHandler) Healthz(c *gin.Context) {
	util.OK(c, gin.H{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"service": "agridispatch",
	})
}

// Readyz 就绪检查（含 DB 与 Redis ping）。
func (h *HealthHandler) Readyz(c *gin.Context) {
	dbStatus := "up"
	if sqlDB, err := h.db.DB(); err != nil || sqlDB.Ping() != nil {
		dbStatus = "down"
	}
	redisStatus := "up"
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "down"
	}
	status := "ok"
	if dbStatus != "up" || redisStatus != "up" {
		status = "degraded"
	}
	util.OK(c, gin.H{
		"status":  status,
		"db":      dbStatus,
		"redis":   redisStatus,
		"time":    time.Now().Format(time.RFC3339),
		"service": "agridispatch",
	})
}

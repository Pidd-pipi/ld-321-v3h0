package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/model"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/redis/go-redis/v9"
)

// DashboardService 调度看板服务（带 Redis 缓存）。
type DashboardService struct {
	repo   *repository.DashboardRepository
	redis  *redis.Client
	logger *slog.Logger
}

func NewDashboardService(repo *repository.DashboardRepository, redis *redis.Client, logger *slog.Logger) *DashboardService {
	return &DashboardService{repo: repo, redis: redis, logger: logger}
}

// Overview 返回看板总览（优先读 Redis 缓存，未命中回源 DB 并写缓存）。
func (s *DashboardService) Overview(ctx context.Context) (*model.FarmOverview, error) {
	cached, err := s.redis.Get(ctx, constants.OverviewCacheKey).Result()
	if err == nil {
		var ov model.FarmOverview
		if jsonErr := json.Unmarshal([]byte(cached), &ov); jsonErr == nil {
			return &ov, nil
		}
	}
	ov, err := s.repo.Overview()
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(ov)
	if err := s.redis.Set(ctx, constants.OverviewCacheKey, data, constants.OverviewCacheTTLSeconds*time.Second).Err(); err != nil {
		s.logger.Warn("cache overview failed", "err", err)
	}
	return ov, nil
}

// Invalidate 使缓存失效（派单后调用）。
func (s *DashboardService) Invalidate(ctx context.Context) {
	if err := s.redis.Del(ctx, constants.OverviewCacheKey).Err(); err != nil {
		s.logger.Warn("invalidate overview cache failed", "err", err)
	}
}

// ExportReport 作业报表导出信息。
func (s *DashboardService) ExportReport(ctx context.Context) (map[string]interface{}, error) {
	ov, err := s.Overview(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"fileName": "farm-work-report-2026-05.csv",
		"rows":     len(ov.Records),
		"status":   "ready",
	}, nil
}

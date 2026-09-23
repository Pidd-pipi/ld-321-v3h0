package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/dto"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/redis/go-redis/v9"
)

// DispatchService 任务派单确认与改派服务。
type DispatchService struct {
	repo   *repository.DispatchRepository
	redis  *redis.Client
	logger *slog.Logger
}

// NewDispatchService 构造派单服务。
func NewDispatchService(repo *repository.DispatchRepository, redis *redis.Client, logger *slog.Logger) *DispatchService {
	return &DispatchService{repo: repo, redis: redis, logger: logger}
}

// invalidateOverviewCache 派单/改派成功后失效看板缓存；Redis 不可用时仅告警不阻断主流程。
func (s *DispatchService) invalidateOverviewCache(ctx context.Context) {
	if s.redis == nil {
		return
	}
	if err := s.redis.Del(ctx, constants.OverviewCacheKey).Err(); err != nil {
		s.logger.Warn("invalidate overview cache failed", "err", err)
	}
}

// ConfirmDispatch 派单确认：请求可指定农机和驾驶员，未传时沿用推荐资源。
// 派单条件不满足时返回明确错误，任务与资源状态均不变（仓储事务保证）。
func (s *DispatchService) ConfirmDispatch(ctx context.Context, taskID string, req dto.DispatchConfirmRequest) (*dto.DispatchResult, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = constants.DefaultDispatchReason
	}
	machineCode, driverName, err := s.repo.ConfirmDispatch(taskID, repository.DispatchParams{
		MachineCode: strings.TrimSpace(req.MachineCode),
		DriverName:  strings.TrimSpace(req.DriverName),
		Reason:      reason,
	})
	if err != nil {
		return nil, fmt.Errorf("confirm dispatch task %s: %w", taskID, err)
	}
	s.invalidateOverviewCache(ctx)
	if s.logger != nil {
		s.logger.Info("task dispatch confirmed",
			"taskId", taskID, "machine", machineCode, "driver", driverName, "reason", reason)
	}
	return &dto.DispatchResult{
		TaskID:      taskID,
		Status:      constants.TaskDispatched,
		MachineCode: machineCode,
		DriverName:  driverName,
		Action:      constants.DispatchActionConfirm,
		Reason:      reason,
		Message:     constants.MsgDispatchConfirmed,
	}, nil
}

// Reassign 已派单任务带原因改派：释放旧资源并原子分配新资源。
// 条件不满足时返回明确错误，任务与资源状态均不变（仓储事务保证）。
func (s *DispatchService) Reassign(ctx context.Context, taskID string, req dto.ReassignRequest) (*dto.DispatchResult, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, fmt.Errorf("%s", constants.ErrMsgReassignReasonBlank)
	}
	machineCode, driverName, err := s.repo.Reassign(taskID, repository.ReassignParams{
		MachineCode: strings.TrimSpace(req.MachineCode),
		DriverName:  strings.TrimSpace(req.DriverName),
		Reason:      reason,
	})
	if err != nil {
		return nil, fmt.Errorf("reassign task %s: %w", taskID, err)
	}
	s.invalidateOverviewCache(ctx)
	if s.logger != nil {
		s.logger.Info("task reassigned",
			"taskId", taskID, "machine", machineCode, "driver", driverName, "reason", reason)
	}
	return &dto.DispatchResult{
		TaskID:      taskID,
		Status:      constants.TaskDispatched,
		MachineCode: machineCode,
		DriverName:  driverName,
		Action:      constants.DispatchActionReassign,
		Reason:      reason,
		Message:     constants.MsgReassigned,
	}, nil
}

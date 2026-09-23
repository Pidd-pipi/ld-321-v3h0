import { API_BASE } from '../constants/app.constants';
import { AppException } from '../errors/AppException';
import { logger } from '../logger/logger';
import type { DashboardItem, DispatchRequest, FarmOverview } from '../types/domain';

export const fetchFarmOverview = async (): Promise<FarmOverview> => {
  const response = await fetch(`${API_BASE}/dashboard/overview`);
  if (!response.ok) {
    logger.error('overview request failed', response.status);
    throw new AppException('OVERVIEW_FAILED', '无法加载农机调度看板数据');
  }
  const body = await response.json();
  // 解包后端统一响应 {code, message, data}
  if (body && typeof body === 'object' && body.code === 0 && body.data) {
    return body.data as FarmOverview;
  }
  return body as FarmOverview;
};

// DispatchResult 派单/改派接口返回。
export interface DispatchResult {
  taskId: string;
  status: string;
  machineCode: string;
  driverName: string;
  reason: string;
  action: string;
  message: string;
}

const postDispatch = async (path: string, payload?: DispatchRequest): Promise<DispatchResult> => {
  const response = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload ?? {}),
  });
  const body = await response.json().catch(() => null);
  if (!response.ok || !body || body.code !== 0) {
    // 后端失败时返回明确中文错误信息，直接展示给调度员
    const message = body && body.message ? body.message : '派单失败，请稍后重试';
    throw new AppException('DISPATCH_FAILED', message);
  }
  return body.data as DispatchResult;
};

// 派单确认：machineCode/driverName 未传时后端沿用任务推荐资源。
export const confirmTask = async (taskId: string, payload?: DispatchRequest): Promise<DispatchResult> =>
  postDispatch(`${API_BASE}/tasks/${taskId}/dispatch`, payload);

// 已派单任务改派：必须携带原因。
export const reassignTask = async (taskId: string, payload: DispatchRequest): Promise<DispatchResult> =>
  postDispatch(`${API_BASE}/tasks/${taskId}/reassign`, payload);

export const saveItems = (items: DashboardItem[]) => {
  localStorage.setItem('agridispatch.items', JSON.stringify(items));
};

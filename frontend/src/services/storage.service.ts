import { API_BASE } from '../constants/app.constants';
import { AppException } from '../errors/AppException';
import { logger } from '../logger/logger';
import type { DashboardItem, DispatchRequestBody, DispatchResult, FarmOverview } from '../types/domain';

interface ApiEnvelope<T> {
  code: number;
  message: string;
  data?: T;
}

const readError = async (response: Response, fallback: string): Promise<AppException> => {
  let message = fallback;
  try {
    const body = (await response.json()) as ApiEnvelope<unknown>;
    if (body && typeof body.message === 'string' && body.message) {
      message = body.message;
    }
  } catch (parseErr) {
    logger.warn('parse error body failed', parseErr);
  }
  return new AppException('DISPATCH_FAILED', message);
};

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

const postTaskAction = async (
  taskId: string,
  action: 'dispatch' | 'reassign',
  payload: DispatchRequestBody,
  fallback: string,
): Promise<DispatchResult> => {
  const response = await fetch(`${API_BASE}/tasks/${taskId}/${action}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw await readError(response, fallback);
  }
  const body = (await response.json()) as ApiEnvelope<DispatchResult>;
  if (body && typeof body === 'object' && body.code === 0 && body.data) {
    return body.data;
  }
  return body as unknown as DispatchResult;
};

// 派单确认：可指定农机编号/驾驶员，不传则沿用任务推荐资源。
export const dispatchTask = async (taskId: string, body: DispatchRequestBody = {}): Promise<DispatchResult> =>
  postTaskAction(taskId, 'dispatch', body, '派单失败');

// 已派单任务改派：必须携带改派原因。
export const reassignTask = async (taskId: string, body: DispatchRequestBody): Promise<DispatchResult> =>
  postTaskAction(taskId, 'reassign', body, '改派失败');

export const saveItems = (items: DashboardItem[]) => {
  localStorage.setItem('agridispatch.items', JSON.stringify(items));
};

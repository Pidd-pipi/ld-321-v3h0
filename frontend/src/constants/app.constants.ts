export const APP_NAME = 'AgriDispatch 农机调度';
export const API_BASE = '/api';
export const STATUS_COLORS: Record<string, string> = {
  空闲: 'success',
  作业中: 'warning',
  维修中: 'danger',
  待派单: 'info',
  已派单: 'warning',
  已完成: 'success',
  在岗: 'success',
  可派单: 'primary',
  休息: 'info',
};

// 派单动作
export const DISPATCH_ACTION_CONFIRM = '派单';
export const DISPATCH_ACTION_REASSIGN = '改派';

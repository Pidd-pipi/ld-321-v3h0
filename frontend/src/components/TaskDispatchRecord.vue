<script setup lang="ts">
import { computed } from 'vue';
import { DISPATCH_ACTION_CONFIRM } from '../constants/app.constants';
import type { DispatchRecord } from '../types/domain';

const props = defineProps<{
  records: DispatchRecord[];
  taskId: string;
}>();

// 任务卡片只展示该任务最近一条派单/改派记录。
const latest = computed(() =>
  props.records
    .filter((r) => r.taskId === props.taskId)
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1))[0],
);

// 改派时仅展示发生变化的资源项。
const changedParts = computed(() => {
  const rec = latest.value;
  if (!rec || rec.action === DISPATCH_ACTION_CONFIRM) return '';
  const parts: string[] = [];
  if (rec.previousMachine && rec.previousMachine !== rec.machineCode) {
    parts.push(`农机 ${rec.previousMachine}`);
  }
  if (rec.previousDriver && rec.previousDriver !== rec.driverName) {
    parts.push(`驾驶员 ${rec.previousDriver}`);
  }
  return parts.join(' / ');
});

const formatTime = (value: string) => {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value.replace('T', ' ').slice(0, 16);
  return date.toLocaleString('zh-CN', { hour12: false });
};
</script>

<template>
  <p v-if="latest" class="mt-1 text-xs text-slate-500">
    <el-tag size="small" :type="latest.action === DISPATCH_ACTION_CONFIRM ? 'success' : 'warning'" disable-transitions>
      {{ latest.action }}
    </el-tag>
    <span class="ml-1">{{ latest.machineCode }} / {{ latest.driverName }}</span>
    <span v-if="changedParts" class="text-slate-400">（原 {{ changedParts }}）</span>
    <span class="ml-1">原因：{{ latest.reason }}</span>
    <span class="ml-1 text-slate-400">{{ formatTime(latest.createdAt) }}</span>
  </p>
</template>

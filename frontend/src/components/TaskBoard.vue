<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage } from 'element-plus';
import { STATUS_COLORS } from '../constants/app.constants';
import type { DispatchResult, Driver, FarmTask, Machine } from '../types/domain';
import TaskDispatchDialog from './TaskDispatchDialog.vue';

const props = defineProps<{
  tasks: FarmTask[];
  machines: Machine[];
  drivers: Driver[];
}>();

const emit = defineEmits<{ (e: 'changed'): void }>();

const dialogVisible = ref(false);
const dialogMode = ref<'dispatch' | 'reassign'>('dispatch');
const activeTask = ref<FarmTask | null>(null);

const openDispatch = (task: FarmTask) => {
  activeTask.value = task;
  dialogMode.value = 'dispatch';
  dialogVisible.value = true;
};

const openReassign = (task: FarmTask) => {
  activeTask.value = task;
  dialogMode.value = 'reassign';
  dialogVisible.value = true;
};

// 成功后刷新看板；失败由请求层抛出明确错误，仅弹提示，本地状态不变。
const handleSuccess = (result: DispatchResult) => {
  ElMessage.success(result.message);
  emit('changed');
};

const handleError = (err: unknown) => {
  ElMessage.error(err instanceof Error ? err.message : '操作失败');
};

const formatTime = (iso: string) => {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
};
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">作业任务调度</h2>
    <div class="grid gap-3">
      <article v-for="task in props.tasks" :key="task.id" class="rounded-md border border-slate-200 p-3">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <strong>{{ task.type }} · {{ task.field }}</strong>
              <el-tag :type="STATUS_COLORS[task.status]" size="small">{{ task.status }}</el-tag>
              <el-tag size="small" effect="plain">{{ task.priority }}</el-tag>
            </div>
            <p class="mt-1 text-sm text-slate-600">
              {{ task.areaMu }} 亩 · 预计 {{ task.estimatedHours }} 小时 · {{ task.plannedWindow }}
            </p>
            <p v-if="task.status === '已派单'" class="mt-1 text-sm text-indigo-700">
              已派 {{ task.assignedMachine || task.recommendedMachine }}
              / {{ task.assignedDriver || task.recommendedDriver }}
            </p>
            <p v-else class="mt-1 text-sm text-emerald-700">
              推荐 {{ task.recommendedMachine }} / {{ task.recommendedDriver }}
            </p>
            <!-- 最近派单/改派记录 -->
            <div v-if="task.latestRecord" class="mt-2 rounded bg-slate-50 px-2 py-1.5 text-xs text-slate-500">
              <el-tag size="small" :type="task.latestRecord.action === '改派' ? 'warning' : 'info'">
                {{ task.latestRecord.action }}
              </el-tag>
              <span class="ml-1">{{ task.latestRecord.machineCode }} / {{ task.latestRecord.driverName }}</span>
              <span class="ml-1 text-slate-400">{{ formatTime(task.latestRecord.createdAt) }}</span>
              <p class="mt-1 text-slate-600">原因：{{ task.latestRecord.reason || '—' }}</p>
            </div>
          </div>
          <div class="flex shrink-0 flex-col gap-2">
            <el-button
              v-if="task.status === '待派单'"
              size="small"
              type="primary"
              @click="openDispatch(task)"
            >
              派单确认
            </el-button>
            <el-button
              v-if="task.status === '已派单'"
              size="small"
              type="warning"
              plain
              @click="openReassign(task)"
            >
              改派
            </el-button>
          </div>
        </div>
      </article>
    </div>

    <TaskDispatchDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :task="activeTask"
      :machines="props.machines"
      :drivers="props.drivers"
      @success="handleSuccess"
      @error="handleError"
    />
  </section>
</template>

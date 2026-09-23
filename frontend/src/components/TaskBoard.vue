<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage } from 'element-plus';
import { STATUS_COLORS } from '../constants/app.constants';
import type { DispatchRecord, Driver, FarmTask, Machine } from '../types/domain';
import DispatchDialog from './DispatchDialog.vue';
import TaskDispatchRecord from './TaskDispatchRecord.vue';

const props = defineProps<{
  tasks: FarmTask[];
  machines: Machine[];
  drivers: Driver[];
  dispatchRecords: DispatchRecord[];
}>();

const emit = defineEmits<{ (e: 'refresh'): void }>();

const dialogVisible = ref(false);
const activeTask = ref<FarmTask | null>(null);

const openDispatch = (task: FarmTask) => {
  activeTask.value = task;
  dialogVisible.value = true;
};

const handleDispatched = () => {
  ElMessage.success('看板数据已刷新');
  emit('refresh');
};
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">作业任务调度</h2>
    <div class="grid gap-3">
      <article v-for="task in tasks" :key="task.id" class="rounded-md border border-slate-200 p-3">
        <div class="flex items-start justify-between gap-3">
          <div>
            <div class="flex items-center gap-2">
              <strong>{{ task.type }} · {{ task.field }}</strong>
              <el-tag :type="STATUS_COLORS[task.status]" size="small">{{ task.status }}</el-tag>
            </div>
            <p class="mt-1 text-sm text-slate-600">
              {{ task.areaMu }} 亩 · 预计 {{ task.estimatedHours }} 小时 · {{ task.plannedWindow }}
            </p>
            <p class="mt-1 text-sm text-emerald-700">
              推荐 {{ task.recommendedMachine }} / {{ task.recommendedDriver }}
            </p>
            <p v-if="task.status === '已派单'" class="mt-1 text-sm text-amber-700">
              已分配 {{ task.assignedMachine }} / {{ task.assignedDriver }}
            </p>
            <TaskDispatchRecord :records="dispatchRecords" :task-id="task.id" />
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
              v-else-if="task.status === '已派单'"
              size="small"
              type="warning"
              @click="openDispatch(task)"
            >
              改派
            </el-button>
            <el-tag v-else size="small" type="info">不可操作</el-tag>
          </div>
        </div>
      </article>
    </div>

    <DispatchDialog
      v-model:visible="dialogVisible"
      :task="activeTask"
      :machines="props.machines"
      :drivers="props.drivers"
      @dispatched="handleDispatched"
    />
  </section>
</template>

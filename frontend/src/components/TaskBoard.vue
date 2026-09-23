<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { STATUS_COLORS } from '../constants/app.constants';
import { dispatchTask } from '../services/storage.service';
import type { FarmTask } from '../types/domain';

defineProps<{ tasks: FarmTask[] }>();

const handleDispatch = async (task: FarmTask) => {
  const result = await dispatchTask(task.id);
  ElMessage.success(result.message);
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
          </div>
          <el-button size="small" type="primary" @click="handleDispatch(task)">一键派单</el-button>
        </div>
      </article>
    </div>
  </section>
</template>

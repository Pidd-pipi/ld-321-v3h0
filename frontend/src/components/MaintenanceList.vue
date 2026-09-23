<script setup lang="ts">
import type { MaintenanceReminder } from '../types/domain';

defineProps<{ reminders: MaintenanceReminder[] }>();
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-black">维修保养提醒</h2>
    <el-timeline>
      <el-timeline-item
        v-for="item in reminders"
        :key="item.id"
        :type="item.level === 'danger' ? 'danger' : item.level === 'warning' ? 'warning' : 'success'"
        :timestamp="item.dueDate"
      >
        <strong>{{ item.machineCode }} · {{ item.title }}</strong>
        <p class="text-sm text-slate-600">剩余 {{ item.remainingHours }} 小时 · {{ item.lastServiceRecord }}</p>
      </el-timeline-item>
    </el-timeline>
  </section>
</template>

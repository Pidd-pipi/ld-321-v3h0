<script setup lang="ts">
import { STATUS_COLORS } from '../constants/app.constants';
import type { Machine } from '../types/domain';

defineProps<{ machines: Machine[] }>();
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-lg font-black">农机档案</h2>
      <el-select placeholder="按状态筛选" size="small" class="w-36">
        <el-option label="全部" value="all" />
        <el-option label="空闲" value="idle" />
        <el-option label="作业中" value="working" />
        <el-option label="维修中" value="repair" />
      </el-select>
    </div>
    <el-table :data="machines" size="small">
      <el-table-column prop="code" label="编号" width="120" />
      <el-table-column prop="name" label="农机" />
      <el-table-column prop="model" label="型号" width="110" />
      <el-table-column prop="horsepower" label="马力" width="80" />
      <el-table-column prop="field" label="所属地块" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="STATUS_COLORS[row.status]" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="qrCode" label="二维码" width="110" />
    </el-table>
  </section>
</template>

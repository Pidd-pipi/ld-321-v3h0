<script setup lang="ts">
import type { TrackPoint } from '../types/domain';

defineProps<{ tracks: TrackPoint[] }>();
</script>

<template>
  <section class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-lg font-black">地图与轨迹数据</h2>
      <el-tag>高德 JSAPI 数据结构</el-tag>
    </div>
    <div class="grid min-h-56 place-items-center rounded-lg bg-emerald-950 p-4 text-white">
      <div class="w-full">
        <div class="mb-4 h-28 rounded border border-emerald-400/40 bg-emerald-900/80 p-3">
          <div v-for="point in tracks" :key="point.capturedAt" class="text-sm">
            {{ point.capturedAt }} {{ point.machineCode }} {{ point.longitude }}, {{ point.latitude }}
          </div>
        </div>
        <p class="text-sm text-emerald-100">
          轨迹按作业类型着色，支持日期回放、缩放和地块边界绘制；本 demo 暴露后端轨迹坐标供地图组件接入。
        </p>
      </div>
    </div>
  </section>
</template>

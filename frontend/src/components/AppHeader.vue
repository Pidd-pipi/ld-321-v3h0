<script setup lang="ts">
import { APP_NAME } from '../constants/app.constants';
</script>

<template>
  <header class="border-b border-emerald-900/10 bg-white">
    <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
      <div>
        <p class="text-sm font-semibold text-emerald-700">农业合作社作业中心</p>
        <h1 class="text-2xl font-black">{{ APP_NAME }}</h1>
      </div>
      <el-tag type="success" effect="dark">WebSocket 轨迹演示</el-tag>
    </div>
  </header>
</template>

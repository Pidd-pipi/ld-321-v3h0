<script setup lang="ts">
import type { DashboardItem } from '../types/domain';

defineProps<{ item: DashboardItem }>();
defineEmits<{ (event: 'advance', id: string): void }>();
</script>

<template>
  <article class="card p-4">
    <div class="flex justify-between gap-3">
      <div>
        <h3 class="text-lg font-black">{{ item.title }}</h3>
        <p class="mt-2 text-sm leading-6 text-stone-600">{{ item.description }}</p>
      </div>
      <span class="font-bold text-field">{{ item.score }}</span>
    </div>
    <div class="mt-4 flex items-center justify-between">
      <span class="text-sm text-stone-600">{{ item.status }}</span>
      <button class="rounded-md bg-field px-3 py-2 text-sm font-bold text-white" @click="$emit('advance', item.id)">推进</button>
    </div>
  </article>
</template>

<script setup lang="ts">
defineProps<{ label: string; value: string; note: string }>();
</script>

<template>
  <article class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
    <p class="text-sm text-slate-500">{{ label }}</p>
    <strong class="mt-2 block text-3xl font-black text-slate-950">{{ value }}</strong>
    <span class="mt-1 block text-sm text-slate-600">{{ note }}</span>
  </article>
</template>

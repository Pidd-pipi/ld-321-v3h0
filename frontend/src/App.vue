<script setup lang="ts">
import DashboardView from './features/DashboardView.vue';
import AppHeader from './components/AppHeader.vue';
</script>

<template>
  <main class="min-h-screen bg-slate-100 text-slate-900">
    <AppHeader />
    <DashboardView />
  </main>
</template>

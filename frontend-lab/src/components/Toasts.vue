<script setup>
import { app } from "@/store/app";
import Icon from "./Icon.vue";
</script>

<template>
  <div class="toasts" aria-live="polite">
    <div v-for="t in app.toasts" :key="t.id" class="toast" :class="{ w: t.kind === 'err' }">
      <Icon :name="t.kind === 'err' ? 'warning' : 'check'" /><span>{{ t.text }}</span>
    </div>
  </div>
</template>

<script setup>
// Marca do filebrowser-lab: disco escuro com pasta laranja → rosa e barra de uso ciano.
defineProps({ size: { type: Number, default: 38 }, text: { type: Boolean, default: true } });
const id = "lg" + Math.random().toString(36).slice(2, 7);
</script>

<template>
  <span class="logo">
    <svg :width="size" :height="size" viewBox="0 0 38 38" aria-hidden="true">
      <defs>
        <linearGradient :id="id + 'a'" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff9630" /><stop offset="1" stop-color="#ff1bca" /></linearGradient>
        <linearGradient :id="id + 'b'" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#01ced3" /><stop offset="1" stop-color="#065e9e" /></linearGradient>
      </defs>
      <rect x="3" y="4" width="32" height="30" rx="8" fill="#1c0d46" stroke="#2f1a67" stroke-width="1.5" />
      <path d="M9 14a2.5 2.5 0 0 1 2.5-2.5h4l2.2 2.2H26.5A2.5 2.5 0 0 1 29 16.2V24a2.5 2.5 0 0 1-2.5 2.5h-15A2.5 2.5 0 0 1 9 24z" :fill="'url(#' + id + 'a)'" />
      <rect x="8" y="30" width="22" height="2.2" rx="1.1" :fill="'url(#' + id + 'b)'" />
    </svg>
    <span v-if="text" class="lt"><b>file<i>browser</i></b><small>seu servidor de arquivos</small></span>
  </span>
</template>

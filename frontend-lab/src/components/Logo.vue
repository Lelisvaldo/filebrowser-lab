<script setup>
// Marca do filebrowser-lab: disco escuro com pasta laranja → rosa e barra de uso ciano.
defineProps({ size: { type: Number, default: 46 }, text: { type: Boolean, default: true } });
const id = "lg" + Math.random().toString(36).slice(2, 7);
</script>

<template>
  <span class="logo">
    <svg :width="size" :height="size" viewBox="0 0 38 38" aria-hidden="true">
      <defs>
        <linearGradient :id="id + 'a'" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff9630" /><stop offset="1" stop-color="#ff1bca" /></linearGradient>
        <linearGradient :id="id + 'b'" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#01ced3" /><stop offset="1" stop-color="#065e9e" /></linearGradient>
      </defs>
      <rect x="3" y="4" width="32" height="30" rx="8" fill="#1c0d46" stroke="#3a2480" stroke-width="1" />
      <path d="M9 13a2.5 2.5 0 0 1 2.5-2.5h4l2.2 2.2H26.5A2.5 2.5 0 0 1 29 15.2V22a2.5 2.5 0 0 1-2.5 2.5h-15A2.5 2.5 0 0 1 9 22z" :fill="'url(#' + id + 'a)'" />
      <circle cx="10.1" cy="27.6" r="1.1" fill="#ff9630" />
      <circle cx="13" cy="27.6" r="1.1" fill="#ff15d2" />
      <circle cx="15.9" cy="27.6" r="1.1" fill="#7a76d4" />
      <circle cx="18.8" cy="27.6" r="1.1" fill="#01ced3" />
      <rect x="20.9" y="26.5" width="8.1" height="2.2" rx="1.1" :fill="'url(#' + id + 'b)'" />
    </svg>
    <span v-if="text" class="lt"><b>file<i>browser</i></b><small>seu servidor de arquivos</small></span>
  </span>
</template>

<script setup>
import { computed } from "vue";

// Rosca de uso do disco, com a porcentagem no meio (como na prévia).
const props = defineProps({
  pct: { type: Number, default: 0 },
  index: { type: Number, default: 0 },
  mini: { type: Boolean, default: false },
  warn: { type: Boolean, default: false },
});
const GRADS = ["gdA", "gdB", "gdC", "gdB"];
const RINGS = ["#14d4dc", "#00c6cf", "#ff15d2", "#00c6cf"];
const grad = computed(() => (props.warn ? "gdC" : GRADS[props.index % GRADS.length]));
const ring = computed(() => (props.warn ? "#ff15d2" : RINGS[props.index % RINGS.length]));
const dash = computed(() => ((props.pct / 100) * 213.63).toFixed(1));
</script>

<template>
  <svg :class="mini ? 'mini' : 'big'" viewBox="0 0 100 100" role="img" :aria-label="pct + '% usado'">
    <circle cx="50" cy="50" r="46" fill="none" :stroke="ring" stroke-width="3" />
    <circle cx="50" cy="50" r="34" fill="none" stroke="var(--track)" stroke-width="15" />
    <circle cx="50" cy="50" r="34" fill="none" :stroke="'url(#' + grad + ')'" stroke-width="15"
      :stroke-dasharray="dash + ' 213.63'" transform="rotate(-90 50 50)" />
    <circle cx="50" cy="50" r="24" fill="var(--dkin)" />
    <text x="50" y="58" text-anchor="middle" font-size="25" font-weight="600" fill="var(--txt)">{{ pct }}%</text>
  </svg>
</template>
